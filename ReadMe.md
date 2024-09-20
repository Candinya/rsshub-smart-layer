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

## Configuration

Configuration is combined of 4 parts: system, rsshub, translate and image_proxy. An example can be referred from `config.yml.example`.

1. `system` part defines the basic info of this application, every field is required.
2. `rsshub` part defines all RSSHub instance and their corresponding preferences (preferred platforms and whether it can work as a fallback instance). 
    Different RSSHub has different configuration options, so offload different requests to different instances effectively should be helpful. 
    Or if just one single RSSHub instance is provided, skip `platforms` field and set `fallback` to `true` to handle all incoming requests.  
3. `translate` defines the service provider and other request details. We are using subdomain to identify target language to provide a smooth experience for end users,
    which is configured by `host_base`: in example configuration, our translate-enabled domain is `*.rsl.localhost`. 
    For example, if we request `zh.rsl.locahost`, then `zh` will be select as target language.
    Different translate provider has different settings, for `libretranslate` we are using YAML format. Please refer to different provider settings.
4. `image_proxy` provides a simple image proxy service to bypass image protect mechanisms. To provide more flexibility, we don't pre-define any built-in rules here,
   please add your own rules for different platforms.

Only `system` and `rsshub` parts are required, if you don't want `translate` or `image_proxy` function, simply delete them.

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
