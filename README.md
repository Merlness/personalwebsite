# Personal Website

![Tests](https://github.com/Merlness/personalwebsite/actions/workflows/test.yml/badge.svg)

A sophisticated personal website built with Go, Templ, and Tailwind CSS.

## Features

- **Home**: Elegant intro page.
- **Portfolio**: Showcase for Landscape, Wildlife, and Portrait photography.
- **Blog**: Journal for stories and trips.
- **About**: Profile and contact links.
- **Design**: Black and Silver aesthetic.

## Development

### Prerequisites

- Go 1.23+
- Node.js & npm (for Tailwind)

### Commands

- `make run`: Generate templates, build CSS, and start the server.
- `make test`: Run the test suite.
- `make build-css`: Rebuild Tailwind CSS.
- `make generate`: Regenerate Templ components.

## Architecture

- **HTMX/Templ**: Server-side rendering with strongly typed components.
- **Tailwind CSS**: Utility-first styling.
- **Standard Library**: Uses Go's `net/http` with new 1.22+ routing patterns.
- **Domain Driven**: Organized by feature (blog, web, etc.).

## adding Content

### Blog Posts
Modify `internal/blog/service.go` to add new posts to the `memoryService` struct.
To make it more dynamic, you can implement a new `Service` that reads Markdown files.

### Portfolio
Modify `internal/web/components/portfolio.templ` to add new images and categories.
