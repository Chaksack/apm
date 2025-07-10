# APM Documentation

This directory contains the documentation for APM (Application Performance Monitoring for GoFiber).

## Viewing the Documentation

### Online (GitHub Pages)

Visit [https://chaksack.github.io/apm](https://chaksack.github.io/apm) to view the documentation online.

### Local Development

To run the documentation locally:

1. Install Jekyll:
   ```bash
   gem install bundler jekyll
   ```

2. Install dependencies:
   ```bash
   cd docs
   bundle install
   ```

3. Run the development server:
   ```bash
   bundle exec jekyll serve
   ```

4. Open http://localhost:4000/apm in your browser

## Documentation Structure

- `index.md` - Main documentation homepage
- `quickstart.md` - Quick start guide
- `cli-reference.md` - CLI command reference
- `features.md` - Comprehensive feature overview
- `_config.yml` - Jekyll configuration
- `_layouts/` - Custom page layouts

## Contributing to Documentation

1. Edit or create Markdown files in this directory
2. Follow the existing format with YAML front matter
3. Test locally before submitting PR
4. Documentation automatically deploys via GitHub Pages

## Enabling GitHub Pages

To enable GitHub Pages for this repository:

1. Go to Settings → Pages
2. Source: Deploy from a branch
3. Branch: main (or your default branch)
4. Folder: /docs
5. Save and wait for deployment

The documentation will be available at:
`https://[your-username].github.io/apm`