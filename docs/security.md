# Güvenlik

## Tehdit sınırı

Korunan varlıklar: yönetim verisi, imza anahtarı, düğüm tokenı ve public trafiğin gideceği backend adresidir. Saldırgan public dinleyiciye, çalınmış bir panel tokenına veya sahte bir konfigürasyon paketine sahip olabilir.

Bu sürümün karşılıkları:

- Paket Ed25519 ile imzalanır. Bozuk, imzasız veya hash'i değişmiş paket uygulanmaz.
- Joker hostname ve ham Envoy YAML/Lua girişi yoktur.
- Bilinmeyen hostname için route üretilmez.
- Uç nokta adresi olarak loopback, link-local ve `metadata.google.internal` reddedilir. `10.0.0.0/8` gibi özel adresler backend olabildiği için reddedilmez.
- DNS rebinding ve yönlendirme sonrası hedef kontrolü test edilmedi.
- Viewer rolü yazma isteklerinde 403 alır. Agent tokenı yönetim CRUD'una giremez.
- API gövdesi 1 MiB ile sınırlıdır. Rol başına saniyede 100 istek sınırı vardır; bu tek süreç içindir.
- İstek yolu normalize edilir, çift eğik çizgi birleştirilir, kaçmış eğik çizgi reddedilir.
- Retry politikası üretilmez. Devre kesici retry bütçesi 0'dır.
- POST için otomatik tekrar yoktur.
- Yanıtlarda imza özel anahtarı ve token yoktur.
- HSTS, JWT, OIDC, CSRF çerezi, WAF ve dış yetkilendirme yoktur.

## Açılış bağımlılığı

Agent, açık anahtar dosyası olmadan doğrulama yapamaz ve bu dosya yerel diskte durmalıdır. Merkez veya bir KMS her açılışta zorunluysa bağımsız açılış sağlanmış sayılmaz. Geliştirmede açık anahtar `.local` içinden bağlanır.

Süresi dolmuş sertifika bu sürümde yoktur çünkü public TLS yoktur. Bağlantısı kopmuş bir düğüm yeni iptal bilgisi alamaz; bu sınır, sertifika ve düğüm iptali gelince servis bazında süre ile tanımlanacak.

## Denetim

Audit satırı aktör, eylem, hedef, önce/sonra ve sonuç tutar. Satırın veritabanında durması değiştirilemezlik değildir. Ayrı yetkili ve konum dışı hedef yoktur. Audit yazılamazsa işlem geri alınır; bu, trafik düğümünü durdurmaz.

## Kenar

Envoy admin yalnızca konteyner içinde `127.0.0.1:9901` dinler ve yayınlanmaz. PostgreSQL Compose ağı dışında yayınlanmaz. Bu, üretimde veritabanını public yapmak için gerekçe değildir.

X-Forwarded-For güveni tanımlı değildir; header client IP'si sayılmaz. Spoofing testi çalıştırılmadı.
