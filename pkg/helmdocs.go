package pkg

import (
	"errors"
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

func ParseHelmDocsComment(s string) (HelmDocsComment, error) {
	return HelmDocsComment{}, errors.New("not implemented")
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
