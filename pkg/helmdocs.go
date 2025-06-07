package pkg

import (
	"fmt"
	"regexp"
	"strings"
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
var helmDocsCommentRegexp = regexp.MustCompile(`^#\s+(?P<path>(?:\w[^\s\.]*)(?:\.(?:\S+|"[^"]*"))*)?\s*--\s*(?:\((?P<type>[\w/\.-]+)\)\s*)?(?P<desc>.*)`)

type HelmDocsComment struct {
	Path         []string // Example: "# myPath.foo.bar -- My description"
	Description  string   // Example: "# -- My description"
	Type         string   // Example: "# -- (myType) My description"
	NotationType string   // Example: "# @notationType -- myType"
	Default      string   // Example: "# @default -- myDefault"
	Section      string   // Example: "# @section -- mySection"

	CommentsAbove []string // lines above the first "# -- My description"
}

func ParseHelmDocsComment(headComment string) (HelmDocsComment, error) {
	commentsAbove, helmDocsComments := splitHeadCommentsByHelmDocs(headComment)
	helmDocs := HelmDocsComment{}
	if len(commentsAbove) > 0 {
		helmDocs.CommentsAbove = commentsAbove
	}

	if len(helmDocsComments) == 0 {
		return helmDocs, nil
	}

	firstLine := helmDocsComments[0]
	groups := helmDocsCommentRegexp.FindStringSubmatch(firstLine)
	pathString := groups[1]
	helmDocs.Type = groups[2]
	descriptionLines := []string{groups[3]}

	if pathString != "" {
		// TODO: implement proper parsing of path
		helmDocs.Path = append(helmDocs.Path, pathString)
	}

	for _, line := range helmDocsComments[1:] {
		if _, ok := cutSchemaComment(line); ok {
			return HelmDocsComment{}, fmt.Errorf(
				"'# @schema' comments are not supported in helm-docs comments. " +
					"Please set the '# @schema' comment above the helm-docs '# --' line; " +
					"alternatively put it on the line as the YAML key or as a foot comment on the YAML key. " +
					"See the helm-values-schema-json docs for more information: https://github.com/losisin/helm-values-schema-json/blob/main/docs/README.md")
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

// splitHeadCommentsByHelmDocs will split a head comment by line and return:
//
// - Lines from last comment block, up until any helm-docs comments
// - Liens from helm-docs comments
func splitHeadCommentsByHelmDocs(headComment string) (schemaComments, helmDocs []string) {
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
			return comments[:i], comments[i:]
		}
	}
	return comments, nil
}
