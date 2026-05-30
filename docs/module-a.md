# Module A: Framework Recognition and Resource Discovery

## Scope

Module A covers entry HTML fetching, frontend framework/build artifact recognition, source/chunk/source map discovery, manifest/robots/sitemap/OpenAPI resource discovery, and wordlist directory scanning.

It does not own risk tagging, sensitive data judgment, API verification semantics, or final report formatting.

## Inputs

- `targetURL`: the starting page URL.
- `config.Config`: same-origin policy, timeout, concurrency, wordlist path, `MaxResources`, `MaxSourceFiles`, `MaxDepth`, and response preview limits.
- Entry HTML returned by `FetchURL`.
- Source URLs discovered from HTML, JavaScript, source maps, manifests, robots, sitemap, OpenAPI/Swagger, and directory scan resources.
- Built-in and optional local wordlists.

## Outputs

- `model.ResourceRecord`: discovered entry page, JavaScript/module/chunk/source map/manifest/robots/sitemap/OpenAPI-style resources, with source URL, type, tags, frontend recognition, parent URL, and depth.
- `model.SourceFile`: downloaded source text with frontend recognition, related source map paths, restored `sourcesContent`, depth, and parent URL.
- `model.ExtractedCandidate`: API or route clues with source URL, source type, source resource id, discover rule, method hints when available, and recognition tags.
- `BudgetHits`: bounded scan notices such as `max_resources_reached`, `max_source_files_reached`, and `max_depth_reached`.

## Limits

- `MaxResources` limits wordlist-driven directory/resource scan targets.
- `MaxSourceFiles` limits downloaded source/chunk/source map/manifest files.
- `MaxDepth` limits recursive discovery from one source file into nested chunks, maps, or manifest resources. Depth `0` means only initial source URLs are downloaded.
- `SameOrigin` filters cross-origin resource discovery unless explicitly disabled by configuration.
- `Timeout` bounds HTTP requests.

## Analysis Order

1. Fetch entry HTML.
2. Discover script, module preload, source map, JSON, and manifest resources.
3. Detect frontend framework/build artifacts, including Vue, Nuxt, React, Next.js, Angular, Vite, Webpack, and unknown.
4. Download source files recursively within `MaxSourceFiles` and `MaxDepth`.
5. Parse source maps. Prefer `sourcesContent`; otherwise preserve API-like source paths/modules.
6. Parse structured resources first: OpenAPI/Swagger paths, robots rules, sitemap locations, manifest values.
7. Fall back to conservative regex and path heuristics for fetch, axios, XHR, jQuery, JSON/Text, and API-like strings.
