# @litelens/design-system

React 19 component library for litelens — atoms, components, utilities, and hooks for the litelens application and external integrations.

## Table of Contents

- [Version Compatible](#version-compatible)
- [Installation](#installation)
- [Tailwind CSS v4 Setup](#tailwind-css-v4-setup)
- [Usage](#usage)
- [Publishing](#publishing)
- [Contributing](#contributing)

## Version Compatible

| @litelens/design-system | react | tailwindcss |   module   |                remark                |
| :---------------------: | :---: | :---------: | :--------: | :----------------------------------: |
|          v0.x           | ^19.x |    ^4.x     | ECMAScript | TS supported (full `.d.ts` included) |

## Installation

```bash
npm install @litelens/design-system
# or
pnpm add @litelens/design-system
```

## Tailwind CSS v4 Setup

Tailwind CSS v4 is required, via the `@tailwindcss/vite` plugin. After installing the package, `@import` the design system's stylesheet **once** from your app's own CSS entry point (not from JS):

```css
/* src/style.css */
@import "@litelens/design-system/styles.css";

/* your app's own styles below */
```

```tsx
// src/main.tsx or src/index.tsx
import App from "./App";
import "./style.css";

ReactDOM.render(<App />, document.getElementById("root"));
```

This pulls in Tailwind's base layer, `tw-animate-css`, the `shadcn` theme, the Geist Variable font, and all design tokens (`@theme inline` colors, radii, sidebar/chart/success/warning/destructive variables) defined by the design system, so every Tailwind utility class works throughout your app.

## Usage

The package exports from multiple entrypoints for granular imports:

### Atoms (`.`)

Low-level primitive components (buttons, inputs, dialogs, etc.):

```tsx
import { Button } from "@litelens/design-system";
import { Input } from "@litelens/design-system";
import { Badge } from "@litelens/design-system";
```

### Atoms subpath (`./atoms`)

Directly reference the atoms barrel export:

```tsx
import { Button, Checkbox, Select } from "@litelens/design-system/atoms";
```

### Components (`./components`)

Higher-level domain-specific components (resource cells, drawers, modals, etc.):

```tsx
import {
  AnnotationBadge,
  ResourceDetailDrawer,
  ResourceDeletionButton,
} from "@litelens/design-system/components";
```

### Hooks (`./hooks`)

React hooks for common operations:

```tsx
import { useCopyToClipboard } from "@litelens/design-system/hooks";
```

### Utils (`./utils`)

Utility functions for formatting, transformation, and common operations:

```tsx
import { formatRelativeTime, formatTs, cn } from "@litelens/design-system/utils";
```

### Types (`./types`)

TypeScript type definitions for API and navigation:

```tsx
import type { NavItem, NavGroup, UseQueryCallback } from "@litelens/design-system/types";
```

### Libs (`./libs`)

Advanced libraries and utilities (full-text search, etc.):

```tsx
import { FullTextSearchInput, useFullTextSearch } from "@litelens/design-system/libs";
```

## Publishing

Version bumps and publishing are automated:

1. Bump the version in `package.json`
2. Commit and push to the main branch
3. Create a git tag: `git tag design-system/vX.Y.Z` and push it
4. The `.github/workflows/publish-design-system.yml` workflow automatically:
   - Builds the package using `tsup`
   - Publishes to npm
   - Creates a GitHub release

## Contributing

For contribution guidelines and setup instructions, see the [main repository](/).
