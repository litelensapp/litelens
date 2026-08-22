Rendered from [`architecture-light.mmd`](architecture-light.mmd) / [`architecture-dark.mmd`](architecture-dark.mmd)
with `mermaid-cli` — GitHub's mobile apps don't render `mermaid` code fences, so the diagram ships as static
SVGs that switch with the OS/browser color scheme. Regenerate after editing a source:

```
npx @mermaid-js/mermaid-cli -i docs/architecture/architecture-light.mmd -o docs/architecture/architecture-light.svg -b white
npx @mermaid-js/mermaid-cli -i docs/architecture/architecture-dark.mmd -o docs/architecture/architecture-dark.svg -b transparent
```
