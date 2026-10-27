# ADR 0002: Kontrol ve veri düzleminin ayrılması

## Durum

Kabul edildi.

## Karar

Kullanıcı bağlantısı yalnızca Envoy'a gelir. Yönetim API'si ve PostgreSQL, public listener'ın arkasında değildir. Compose'da bu ayrım iki Docker ağıyla durur. Üretimde yönetim erişimi ayrı bir ağ veya eşdeğer erişim yolundadır ve public gateway'in ayakta olmasına bağlı değildir.

API'nin durması, Envoy'un belleğindeki son konfigürasyonu silmez. Agent de diske yazdığı son doğrulanmış paketi, kontrol katmanı yokken yeniden yükleyebilir.

## Bilinçli sınır

Bu oturum tek API süreci ve tek PostgreSQL'dir. Oturum veya iş, süreç belleğine bağlanmaz; kalıcı durum veritabanındadır. Çoklu worker, lease ve fencing Aşama 2'dir. Aynı idempotency anahtarı aynı yayım yanıtını döner. Farklı anahtarla eşzamanlı yayım, satır kilidi sayesinde ikinci istekte sürüm çakışması verir.
