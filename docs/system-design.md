```mermaid
---
title: System Design (JJ GH Top Langs)
---

flowchart TD
  A([User opens GitHub README]) --> B[GitHub Camo<br/>(Image Proxy)]

  B --> C[Server Host<br/>(Go API)]

  C --> D[HTTP Handler]

  D --> E{Cache In-Memory<br/>(SVG only)}

  E -- HIT --> F[Return Cached SVG]

  E -- MISS --> G[Service Layer]

  G --> H[Fetch Repositories<br/>/users/:user/repos]

  H --> I[Fetch Languages per Repo<br/>(sequential)]

  I --> J[Aggregation<br/>(sum bytes per language)]

  J --> K[Calculate Percentages]

  K --> L{Response Type}

  L --> M[Template Engine]

  M --> |index.tmpl| N[Index Page]

  L -->|JSON| O[JSON Encoder]

  M --> |toplangs.tmpl| P[Rendered SVG]

  P --> Q[Store in In-Memory Cache]

  Q --> F

  F --> R([Response to Client <br />e.g. Browser])

  N --> R
  O --> R
```
