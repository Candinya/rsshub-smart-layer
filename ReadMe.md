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
