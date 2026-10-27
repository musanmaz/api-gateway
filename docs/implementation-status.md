# Uygulama durumu

Aşama 0 ve yerel iki düğümlü akış bu oturumdadır. Sonuçlar `docs/test-results.md` dosyasındadır. Orada PASS yazmayan madde tamamlanmış sayılmaz.

## Tamamlanan

- Monorepo, goose migrasyonu, OpenAPI taslağı, Compose
- Servis taslağı, doğrulama, yayım farkı, yayım, audit, idempotency
- Elle geri alma: eski paket yeni revision olarak yeniden imzalanır
- Ed25519 paket, agent diski, ADS
- HTTP, HTTPS, HTTP/2, WebSocket, SSE, gRPC, TCP, UDP ve TLS geçişi ayrı dinleyicilerde
- İki gateway düğümü ve yerel edge. Bir düğüm durunca yeni bağlantı diğerinden gider
- Türkçe panel: 4 menü (Ana sayfa, Servisler, Geçmiş, Sistem), 5 adımlı yayın sihirbazı (adres, hedef, kontrol, yayına al, dene), tek sunucuda deneme ve hepsine açma, sade sunucu durumu, kopyalanabilir curl komutu (`GATEWAY_TRY_URL`). Doğrulama kullanıcı tarafından çalıştırılacak; henüz PASS değil.
- Metrik yokken trafik sağlığı Unknown kalır

## Doğrulanmayan

- Public ACME ve kurumsal sertifika sağlayıcısı. Yalnızca yerel CA var
- OIDC ve ekip/ortam RBAC. Kimlik sağlayıcı yok
- Canary, NACK ve eşzamanlı yayım kodda ve `scripts/verify-e2e.sh` içinde. PASS yazılmadan tamamlanmış sayılmaz. Otomatik rollback yok
- Yük, yedek geri dönüşü, quorum ve lokasyon kaybı

## Sonraki somut adım

Sertifika sağlayıcısı ve kimlik sağlayıcı bilinince public TLS ve OIDC bağlanır. O zamana kadar bu ikisi açılmaz. Üretim VIP'i ve ikinci lokasyon ayrıca kurulmalıdır; yerel edge bunu karşılamaz.
