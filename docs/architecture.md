# Mimari

gateway-control, kullanıcı trafiğini yönetim katmanından ayırır. Tarayıcı panele gider. Panel yalnızca yönetim API'sine konuşur. API PostgreSQL'e yazar. Gateway düğümündeki agent imzalı paketi çeker, diske koyar ve Envoy'a yerel xDS verir. İstemci isteği API'den, veritabanından ve panelden geçmez.

```text
istemci -> Envoy -> backend
panel -> API -> PostgreSQL
agent -> API
agent -> yerel xDS -> Envoy
```

Compose iki ağ kullanır. `mgmt`: panel, API, PostgreSQL, agent. `data`: agent, Envoy, iki örnek backend. Envoy yönetim ağına bağlı değildir.

## Sabit sürümler

2026-10-22 tarihinde resmî kayıtlardan alındı. `latest` ve geliştirme etiketleri yok.

- Envoy `distroless-v1.40.0`, index `sha256:23b72ac1fa94908b2ca7ebf194cc6083cd0d4913210c7d7453e4b2008ca26c84`
- Go `1.27.3`, imaj index `sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5`
- `go-control-plane` `v0.14.1-0.20261013120021-5c5ac772744b` (resmî `envoy/v1.40.0` etiketi) ve API modülü `envoy v1.40.0`. Kök modülün semver `v0.14.0` sürümü Envoy API 1.36 istediği için kullanılmadı.
- PostgreSQL `18.6-bookworm`, index `sha256:bbf83594798e144bff365a9c191c5d779f181fb3135902064ab8ff11918020fc`
- Node.js `24.22.1`, Next.js `16.4.2`, React `19.3.1`, Tailwind CSS `4.3.3`

## Varsayımlar

Trafik hacmi, sunucu sayısı, ağ topolojisi, IdP ve sertifika sağlayıcısı bilinmiyor. Bu yüzden üretim boyutu, HA düğüm sayısı ve sertifika sağlayıcısı kesinleştirilmedi. Yerel varsayılanlar: tek operatör tokenı, düz HTTP, tek gateway düğümü, `local-dev` filosu.

## Dağıtım

İstenen model: API imzalı ve filo kapsamlı paket üretir. Agent dışarı doğru yönetim API'sine bağlanır, Ed25519 imzasını doğrular, paketi atomik dosya değişimiyle saklar ve Unix soket üzerinden ADS sunar. Agent kullanıcı trafiğini taşımaz.

Compose'da soket iki konteyner arasında paylaşılan volume'dadır ve hosta yayınlanmaz. Üretimde soket izinleri ayrı bir grupla daraltılır. Bu oturumda soket modu `0660` ve sahip uid `65532`'dir.

Hostname uç noktaları Envoy `STRICT_DNS` kümesi olarak gider. EDS, IP envanteri olmadan hostname çözmediği için bu dilimde ayrı EDS kaynağı üretilmez. Küme yine CDS ile gelir.

Boş veya doğrulanamayan pakette listener açılmaz. Joker `*` domain üretilmez.

## Zaman

Veritabanı oturumu UTC'dir. Panel varsayılanı `Europe/Istanbul`'dur ve yalnızca `localStorage` içinde durur.
