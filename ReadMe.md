# RSSHub Smart Layer

> An enhancement overlay for RSSHub

## Modules

- Load Balance
- Machine Translate
- Image Proxy

## Workflow

1. Request RSSHub endpoint (through JSON format) till one success, or 503
2. (Optional) Send to machine translate and cache results
3. (Optional) Apply image proxy rules
4. Re-construct feed to target format

## Tech spec

### Load Balance

Based on random numbers.

Maybe add redis based load-balance in the future.

### Translate

#### Supported Translate Providers

- LibreTranslate

#### Add more provider

1. Copy `modules/translate/providers/libretranslate` directory and rename to your target provider
2. Update relative codes (settings, initializer, translate func)
3. Update `modules/translate/providers/new.go` func, add your provider

### Image Proxy

Currently only `Origin` and `Referer` can be specified.

Maybe add more options in the future.

### Response format

Follows RSSHub format query, currently 3 formats:

- `rss`: RSS 2.0 (default / fallback)
- `atom`: Atom
- `json`L JSON Feed
