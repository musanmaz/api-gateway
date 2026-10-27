# ADR 0003: İmzalı paket ve yerel xDS

## Durum

Kabul edildi.

## Karar

PostgreSQL istenen durumun kaynağıdır. Yayım, filo için deterministik bir JSON paket üretir, SHA-256 özeti ve Ed25519 imzası ekler. Revision kimliği monoton artar. İçerik, oluşturan kişi, filo ve önceki revision saklanır.

Agent paketi doğrulamadan Envoy önbelleğine koymaz ve diske yazmaz. Disk yazımı geçici dosya, `fsync` ve `rename` sırasıdır. `current` işaretçisi, kabul edilen dosyadan sonra güncellenir. Bozuk `current`, daha eski doğrulanmış dosyaya döner. Hiçbiri doğrulanamazsa snapshot yüklenmez.

Geliştirme özel anahtarı `.local/keys` içindedir ve yalnızca API konteynerine bağlanır. Agent yalnız açık anahtarı görür. Üretimde özel anahtar işletim sistemi veya gizli alanında durmalıdır. Veritabanındaki imza, özel anahtarın kendisi değildir.

## Ertelenen

SDS, sertifika dağıtımı, düğüm kaydı için mTLS ve filo dışına secret sızdırmama Aşama 1'dir. Şimdiki agent tokenı tek geliştirme düğümünü hedefler; çalınan token başka bir düğüm adı için de paket çekebilir. Bu, üretim kimlik modeli değildir.
