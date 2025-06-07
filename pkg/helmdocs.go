package pkg

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// This became a quite long regexp, but it needs to handle the following special cases:
//
//	# -- A very simple comment
//	#    --    a lot of spacing
//	# -- (string) A very simple comment
//	# --(string)No spacing
//	# -- (tpl/array) Custom type
//	# ------- Dash overload
//	# myField.foobar -- (string) This is my description
//	# myField."foo bar! :D".hello -- (string) This is my description
//	# myField."kubernetes.io/hostname" -- (string) This is my description
//
// If you're having issues grasping the regexp, then this site could help:
// https://regexper.com/#%5E%23%5Cs%2B%28%28%3F%3A%22%5B%5E%22%5D*%22%7C%5B%5E%40%5C-%5Cs%5D*%29%28%3F%3A%5C.%28%3F%3A%22%5B%5E%22%5D*%22%7C%5B%5E%5Cs%5C.%5D*%29%29*%29%3F%5Cs*--%5Cs*%28%3F%3A%5C%28%28%5B%5Cw%2F%5C.-%5D%2B%29%5C%29%5Cs*%29%3F%28.*%29
var helmDocsCommentRegexp = regexp.MustCompile(`^#\s+(?<path>(?:"[^"]*"|[^@\-\s]*)(?:\.(?:"[^"]*"|[^\s\.]*))*)?\s*--\s*(?:\((?<type>[\w/\.-]+)\)\s*)?(?<desc>.*)`)

// This regex only matches 1 segment from a path until the next dot.
// It's meant to be re-run on the substring after each dot until all path segments are found.
var helmDocsPathRegexp = regexp.MustCompile(`^(?:"[^"]*"|(?:[^\s\."]*(?:"[^"]*"|[^\s\."]*)*))`)

type HelmDocsComment struct {
	Path         []string // Example: "# myPath.foo.bar -- My description"
	Description  string   // Example: "# -- My description"
	Type         string   // Example: "# -- (myType) My description"
	NotationType string   // Example: "# @notationType -- myType"
	Default      string   // Example: "# @default -- myDefault"
	Section      string   // Example: "# @section -- mySection"
}

func ParseHelmDocsComment(helmDocsComments []string) (HelmDocsComment, error) {
	helmDocs := HelmDocsComment{}

	if len(helmDocsComments) == 0 {
		return helmDocs, nil
	}

	firstLine := helmDocsComments[0]
	groups := helmDocsCommentRegexp.FindStringSubmatch(firstLine)

	if groups == nil {
		// regexp returns nil on no match
		return helmDocs, nil
	}

	pathString := groups[1]
	helmDocs.Type = groups[2]
	descriptionLines := []string{groups[3]}

	if pathString != "" {
		path, err := ParseHelmDocsPath(pathString)
		if err != nil {
			return HelmDocsComment{}, err
		}
		helmDocs.Path = path
	}

	for _, line := range helmDocsComments[1:] {
		if _, ok := cutSchemaComment(line); ok {
			return HelmDocsComment{}, fmt.Errorf(
				"'# @schema' comments are not supported in helm-docs comments.\n" +
					"\tPlease set the '# @schema' comment above the helm-docs '# --' line;\n" +
					"\talternatively put it on the line as the YAML key or as a foot comment on the YAML key.\n" +
					"\tSee the helm-values-schema-json docs for more information:\n" +
					"\thttps://github.com/losisin/helm-values-schema-json/blob/main/docs/README.md")
		}

		withoutPound := strings.TrimSpace(strings.TrimPrefix(line, "#"))
		annotation, value, ok := strings.Cut(withoutPound, "--")
		if !ok {
			descriptionLines = append(descriptionLines, withoutPound)
			continue
		}
		annotation = strings.TrimSpace(annotation)
		value = strings.TrimSpace(value)
		switch annotation {
		case "@notationType":
			helmDocs.NotationType = value
		case "@default":
			helmDocs.Default = value
		case "@section":
			helmDocs.Section = value
		}
	}

	helmDocs.Description = strings.Join(descriptionLines, " ")

	return helmDocs, nil
}

// ParseHelmDocsPath parses the path part of a helm-docs comment. This has
// some weird parsing logic, but it's created to try replicate the logic
// observed when running helm-docs. We can't just copy or reference their
// implementation due to licensing conflicts (MIT vs GPL v3.0)
//
// Example:
//
//	# some-path.foobar -- This is my description
//
// or also with quoted syntax:
//
//	# labels."kubernetes.io/hostname" -- This is my description
func ParseHelmDocsPath(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}

	firstMatch := helmDocsPathRegexp.FindString(path)
	if firstMatch == "" {
		// all we can say is "invalid syntax", because the Regex may have failed
		// because of a multitude of reasons
		return nil, fmt.Errorf("invalid syntax: %s", path)
	}
	if strings.HasPrefix(firstMatch, "\"") {
		return nil, fmt.Errorf("must not start with a quote: %s", path)
	}

	parts := []string{firstMatch}
	rest := path[len(firstMatch):]

	for rest != "" {
		if !strings.HasPrefix(rest, ".") {
			r, _ := utf8.DecodeRuneInString(rest)
			return nil, fmt.Errorf("expected dot separator, but got %q in: %s", r, path)
		}

		rest = rest[1:]
		if rest == "" {
			return nil, fmt.Errorf("expected value after final dot: %s", path)
		}

		match := helmDocsPathRegexp.FindString(rest)
		if match == "" {
			// all we can say is "invalid syntax", because the Regex may have failed
			// because of a multitude of reasons
			return nil, fmt.Errorf("invalid syntax: %s", path)
		}
		rest = rest[len(match):]

		if strings.HasPrefix(match, "\"") {
			// Remove quotes.
			match = match[1 : len(match)-1]

			// The string could contain quotes inside the path,
			// but we don't want to remove those as helm-docs doesn't seem to remove them either.
			// E.g:
			//	`foo."bar".moo` -> []string{`foo`, `bar`, `moo`}
			//	`foo"bar"moo` -> []string{`foo"bar"moo`}
		}

		parts = append(parts, match)
	}

	return parts, nil
}

// SplitHelmDocsComment will split a head comment by line and return:
//
// - Lines from last comment block, up until any helm-docs comments
// - Liens from helm-docs comments
func SplitHelmDocsComment(headComment string) (before, helmDocs []string) {
	if headComment == "" {
		return nil, nil
	}
	if index := strings.LastIndex(headComment, "\n\n"); index != -1 {
		// Splits after the last "comment group". In other words, given this:
		//
		//	# foo
		//	# bar
		//
		//	# moo
		//	# doo
		//	hello: ""
		//
		// Then only consider the last "# moo" & "# doo" comments
		headComment = headComment[index+2:] // +2 to get rid of the "\n\n"
	}
	comments := strings.Split(headComment, "\n")

	for i, comment := range comments {
		if helmDocsCommentRegexp.MatchString(comment) {
			// Clone second slice so it doesn't get messed up when someone append to the first slice
			return comments[:i], slices.Clone(comments[i:])
		}
	}
	return comments, nil
}
