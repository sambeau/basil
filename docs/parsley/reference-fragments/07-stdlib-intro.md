Parsley has two module namespaces:

- **`@std/`** — Pure Parsley functionality (works without Basil server)
- **`@basil/`** — Server-specific functionality (requires Basil runtime)

Import with the appropriate prefix:

```parsley
import @std/math
let {floor, ceil} = import @std/math

import @basil/api
let {notFound, redirect} = import @basil/api
```
