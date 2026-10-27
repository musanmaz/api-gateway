# ADR 0001: Veri düzlemi Envoy

## Durum

Kabul edildi. 2026-10-22.

## Bağlam

Trafik motoru sıfırdan yazılmayacak. Karşılaştırılan seçenekler Envoy 1.40, Apache APISIX ve HAProxy'dir. Hedef, Linux VM üzerinde L7 yönetimidir. Kubernetes zorunlu değildir.

## Karşılaştırma

Envoy, xDS/ADS ve SDS ile konfigürasyonu süreç yeniden başlatmadan günceller. Yönetim bağlantısı kopunca son geçerli konfigürasyonu bellekte tutar. Ayrı bir agent bu belleği diske yazarsa host yeniden başlatması da karşılanır. Lisans Apache-2.0. Operasyon yüzeyi geniştir; bu proje o yüzeyi kendi derleyicisine hapseder, ham YAML veya Lua kabul etmez.

Apache APISIX yönlendirmeyi etcd ve Nginx/OpenResty üzerinde toplar. Dinamik davranışın bir kısmı Lua'dadır. Ürün, Lua girişini ilk sürümde istemiyor. APISIX'i ikinci bir motor olarak işletmek aynı modeli iki kez doğrulamayı gerektirir.

HAProxy VM üzerinde sade ve olgun bir L4/L7 vekilidir. Yeniden yükleme davranışı anlaşılır. Ancak bu kontrol katmanının istediği standart xDS/SDS akışı HAProxy'nin doğal arayüzü değildir. Data plane API'si ayrı bir entegrasyon olur.

## Karar

Somut bir engel görülmedi. Veri düzlemi Envoy'dur. APISIX ve HAProxy için adaptör yazılmaz.

## Sonuç

xDS tutarlılığı kaynak bazındadır; tüm filonun tek veritabanı işlemi olduğu varsayılmaz. NACK ve kısmi kabul sonraki testlerde ayrıca ele alınacak.
