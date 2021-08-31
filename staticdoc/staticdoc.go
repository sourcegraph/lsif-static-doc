package staticdoc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/lsif/conversion"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/lsif/protocol"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/precise"
)

type Files struct {
	ByPath map[string][]byte
}

var TestingOptions = Options{
	MatchingTags:     nil,
	JSON:             true,
	Markdown:         true,
	MarkdownMetadata: true,
}

type Options struct {
	// MatchingTags is a list of tags (exported, unexported, deprecated, etc.) and if the
	// documentation page or section does not match at least one of them, it will be omitted.
	MatchingTags []protocol.Tag

	// Emit raw JSON representation of pages.
	JSON bool

	// Emit Markdown representation of pages.
	Markdown bool

	// Emit metadata in markdown (search keys, tags, etc.)
	MarkdownMetadata bool
}

func Generate(ctx context.Context, r io.Reader, root string, opt Options) (*Files, error) {
	groupedBundleData, err := conversion.Correlate(ctx, r, root, nil)
	if err != nil {
		return nil, errors.Wrap(err, "conversion.Correlate")
	}
	files := &Files{ByPath: map[string][]byte{}}
	for page := range groupedBundleData.DocumentationPages {
		filePath := page.Tree.PathID
		if filePath == "/index" {
			// rename "/index" in case it is a real page.
			filePath = "/__index"
		} else if filePath == "/" {
			filePath = "/index"
		}

		if opt.JSON {
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			enc.SetIndent("", "  ")
			enc.SetEscapeHTML(false)
			if err := enc.Encode(page.Tree); err != nil {
				return nil, errors.Wrap(err, "Encode")
			}
			files.ByPath[filePath+".json"] = buf.Bytes()
		}

		if opt.Markdown {
			var buf bytes.Buffer
			if err := encodeMarkdown(&buf, page, opt); err != nil {
				return nil, errors.Wrap(err, "encodeMarkdown")
			}
			files.ByPath[filePath+".md"] = buf.Bytes()
		}
	}
	return files, nil
}

func encodeMarkdown(w io.Writer, page *precise.DocumentationPageData, opt Options) error {
	fmt.Fprintf(w, "# %s\n\n", page.Tree.Label)
	if page.Tree.Detail.String() != "" {
		fmt.Fprintf(w, "%s", page.Tree.Detail)
	}

	// Write the "Index" section of the page.
	fmt.Fprintf(w, "## Index\n\n")

	wroteOneSubpage := false
	var writeSubpages func(node *precise.DocumentationNode)
	writeSubpages = func(node *precise.DocumentationNode) {
		if !tagsMatch(opt.MatchingTags, node.Documentation.Tags) {
			return
		}
		for _, child := range node.Children {
			if child.PathID != "" {
				if !wroteOneSubpage {
					fmt.Fprintf(w, "* Subpages\n")
					wroteOneSubpage = true
				}
				relPathID := strings.TrimPrefix(child.PathID, path.Dir(page.Tree.PathID))
				relPathID = strings.TrimPrefix(relPathID, "/")
				fmt.Fprintf(w, "  * [%s](%s.md)\n", strings.TrimPrefix(child.PathID, "/"), relPathID)
			} else {
				writeSubpages(child.Node)
			}
		}
	}
	writeSubpages(page.Tree)

	var writeIndex func(node *precise.DocumentationNode, depth int)
	writeIndex = func(node *precise.DocumentationNode, depth int) {
		if !tagsMatch(opt.MatchingTags, node.Documentation.Tags) {
			return
		}
		if depth != 0 {
			relPathID := strings.TrimPrefix(node.PathID, page.Tree.PathID)
			fmt.Fprintf(w, "%s* [%s](%s)\n", strings.Repeat("    ", depth-1), node.Label, relPathID)
		}
		for _, child := range node.Children {
			if child.Node != nil {
				writeIndex(child.Node, depth+1)
			}
		}
	}
	writeIndex(page.Tree, 0)
	fmt.Fprintf(w, "\n\n")

	// Write the actual content sections of the page.
	var writeSection func(node *precise.DocumentationNode, depth int)
	writeSection = func(node *precise.DocumentationNode, depth int) {
		if !tagsMatch(opt.MatchingTags, node.Documentation.Tags) {
			return
		}
		if depth != 0 {
			hash := node.PathID[strings.Index(node.PathID, "#")+len("#"):]
			fmt.Fprintf(w, "%s <a id=\"%s\" href=\"#%s\">%s</a>\n\n", strings.Repeat("#", depth+1), hash, hash, node.Label)
			if opt.MarkdownMetadata {
				if node.Documentation.SearchKey != "" || len(node.Documentation.Tags) > 0 {
					fmt.Fprintf(w, "```\n")
					if node.Documentation.SearchKey != "" {
						fmt.Fprintf(w, "searchKey: %s\n", node.Documentation.SearchKey)
					}
					if len(node.Documentation.Tags) > 0 {
						fmt.Fprintf(w, "tags: %s\n", node.Documentation.Tags)
					}
					fmt.Fprintf(w, "```\n\n")
				}
			}
			if node.Detail.String() != "" {
				fmt.Fprintf(w, "%s", node.Detail)
			}
		}
		for _, child := range node.Children {
			if child.Node != nil {
				writeSection(child.Node, depth+1)
			}
		}
	}
	writeSection(page.Tree, 0)

	return nil
}

func tagsMatch(want, have []protocol.Tag) bool {
	for _, want := range want {
		got := false
		for _, have := range have {
			if have == want {
				got = true
			}
		}
		if !got {
			return false
		}
	}
	return true
}
