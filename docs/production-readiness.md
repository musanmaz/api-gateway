# Üretime hazırlık

Bu depo üretime hazır değildir. Canlı DNS, güvenlik duvarı, sertifika hesabı veya üretim trafiği değiştirmek için yetki vermez.

## Çalışan

- Taslak servis, doğrulama, yayım farkı ve yayım
- İmzalı revision ve elle paket geri alma
- İki Envoy düğümü, yerel edge, iki örnek backend, host ve path yönlendirmesi
- Agent diski ve merkez kapalıyken yeniden açılış, `docs/test-results.md` içinde gerçekten PASS ise
- Ayrı yönetim ve veri ağı

## Açık risk

- Tek API, tek veritabanı. İki trafik düğümü yalnız tek makinede
- Public ACME yok. Yerel CA ve SDS yalnızca geliştirme içindir
- OIDC, MFA ve ekip/ortam yetkisi yok
- İki düğümlü canary ve NACK kaydı var. Otomatik rollback yok. Elle paket geri alma var; servis satırlarını geri yazmaz
- Yedek, PITR ve geri yükleme tatbikatı yok
- Dış sentetik kontrol, metrik, alarm ve WAF yok
- Agent tokenı düğüme kriptografik olarak bağlı değil
- İkinci lokasyon yok

## Canlıya geçmeden önce

Hedef ağ, IdP, sertifika sağlayıcısı, trafik hacmi ve failure domain bilinmelidir. En az iki trafik düğümü, yönetim için ayrı ağ, Patroni veya eşdeğer PostgreSQL, yedek tatbikatı ve bu depodaki NOT RUN testlerinin ilgili ortamda PASS olması gerekir.
