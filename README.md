# lsif-static-doc

Turn LSIF documentation (in Sourcegraph extension format) into static Markdown.

Today, it is primarily useful for snapshot testing - which means it can sometimes generate information an average consumer would not be interested in (or not in the way they would want.)

Long-term, we'd like for it to also be useful for actually generating API documentation for your repositories as static markdown. If you're interested in that use case, please open an issue to let us know.

## Usage

Be sure to use e.g. the `Markdown Preview Enhanced` extension in VS Code when viewing the generated Markdown. It contains `<a name"foobar"></a>` anchors which VS Code's native markdown preview cannot handle.

## Examples of output

https://github.com/sourcegraph/lsif-static-doc-examples
