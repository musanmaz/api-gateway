# Gateway Control — Ürünleştirme planı
Tarih: 2026-10-27
Durum: Canlı test laboratuvarı, genel erişime hazır üretim ürünü değildir.

## Şu anda tamamlanan ve test edilenler
- Envoy xDS sürümlü yayın, imza doğrulama, ACK/NACK ve rollback temeli.
- Tek gateway alan adı /api/{slug}/, mevcut özel hostlar ve otomatik servis URL'leri.
- API Key (SHA-256 özeti), backend Basic/Bearer/custom header şifreli saklama.
- Required client header varlık kontrolü (değer doğrulaması değil).
- HTTP yerel token-bucket rate limiting: servis başına 0-10000 req/s/node; 429 test edildi.
- Türkçe ve İngilizce seçici: login, menü, servis listesi, özet ve sihirbazın ana bölümleri. Diğer sayfalar kısmi Türkçe.
- İki yerel Envoy node failover testi, birincil ana makinedeki HTTPS gateway.
- WireGuard özel tünel: ai 10.77.44.1, speedtest 10.77.44.2.
- speedtest Docker, Compose ve Envoy v1.40.0 imajı; node2 Compose/bootstrap dosyaları /opt/gateway-node2 altında hazır.

## Kritik eksikler (üretime geçişten önce)
P0 Güvenlik ve dağıtım:
- speedtest agent/worker imajlarını imzalı/private registry üzerinden aktar; signing-public-key, per-node token ve envelope decryption key'i onaylı secrets/KMS akışıyla güvenli yükle. İkinci node şu anda CANLI DEĞİL.
- İkinci node'u fleet'e kaydet, revision/ACK testi yap; Envoy edge health-check havuzuna ekle, iki fiziksel host için failover testi.
- Backend servis IP/DNS'lerini iki data-plane sunucusundan erişilebilir özel ağa taşı; lab backend replika canlı iş servislerini temsil etmez.
- Tek public entrypoint ve yönetim DB tek hata noktalarıdır; ikinci TLS ingress + LB/DNS health failover, HA control plane ve PostgreSQL replikasyonu/backup şart.
- OIDC SSO/MFA + RBAC, per-user audit, kısa ömürlü erişim; tarayıcı sessionStorage'da uzun ömürlü operator token kullanımını kaldır.
- WAF güvenli aşamalı rollout: OWASP CRS (Coraza veya uygun yönetilen WAF) sadece DETECT modundan başlat, yanlış pozitifleri izle, kuralları kademeli BLOCK moduna al. Request-body limitleri ve sanitizasyonu doğrula.
- Key material Vault/KMS/HSM, rotate/scope/revoke, secret backup; xDS paketinin şifreli alanlarının korunması ve node key separation.

P1 Trafik güvenliği ve politika:
- Dağıtık/global rate limiting (Redis veya dedicated rate-limit service); per-consumer ve per-route quota, burst, 429 Retry-After, bypass kuralları.
- CIDR allow/deny, trusted X-Forwarded-For chain ve gerçek client IP; geo/bot ve mTLS opsiyonları.
- JWT/OIDC doğrulama, OAuth2 client-credentials token exchange, authz policies (method/path/scope), API key versioning.
- Per-route/per-method inbound/outbound auth rules; required header value/pattern validation ve header rewrite allowlist.
- WAF CRS rule management, score ve istisna UI; yapılandırma diff, audit, rollback.

P2 Operasyon ve gözlemlenebilirlik:
- RPS, 4xx/5xx, p50/p95/p99, 429, WAF blocked, upstream health, circuit breaker, per-service SLO.
- Prometheus, OpenTelemetry/trace ve merkezi yapılandırılabilir log retention; sensitive header redaction.
- Test/preview/dry-run, config canary rollout, node bazlı durum, health checks, zero-downtime publish.
- Kullanıcı dil paketi bütün ekranlarda TR/EN, server-side tercih ve i18n QA.

P3 Ürün/yaşam döngüsü:
- Multi-tenant team/workspace, RBAC, OpenAPI/AsyncAPI import ve developer portal, API lifecycle/versioning/deprecation.
- CI/CD signed container images, SBOM, vulnerability/license scanning, backup/restore drills, compatibility matrix.
- Kurulum/upgrade/rollback belgeleri, policy testleri, yük/chaos testleri ve release süreçleri.

## Sınırlar
- Mevcut RPS kotası Envoy işlemi başına yereldir, küresel değildir. İki Envoy toplam throughput yaklaşık iki kat kota olabilir.
- Rate limiter WAF değildir; SQL injection/XSS koruması sunduğu varsayılamaz.
- WireGuard bağlantısı tek başına HA değildir; remote data-plane ve public ingress failover ayrıca doğrulanmalıdır.
- Prod dağıtımından önce OWASP API Top 10 ve PCI gereksinimlerine göre threat model/test raporu hazırlanmalıdır.
