# gateway-control

Web siteleri ve HTTP API'leri için yönetim paneli, kontrol API'si ve Envoy veri düzlemi. Bu depo Aşama 0'dır: yerel Docker Compose üzerinde gerçek HTTP trafiği geçer. Üretim kurulumu değildir.

## Başlatma

Docker ve OpenSSL gerekir. Hostta Go veya Node kurulu olması gerekmez.

```sh
make up
make token
```

`make token` operatör tokenını yazdırır. Arayüz [http://127.0.0.1:3000/login](http://127.0.0.1:3000/login) adresindedir. Token `sessionStorage` içine konur; repoda sabit parola yoktur.

| Uç | Adres |
| --- | --- |
| Panel | http://127.0.0.1:3000 |
| Yönetim API | http://127.0.0.1:9080 |
| Yerel edge | http://127.0.0.1:10000 |
| Gateway düğümü 1 | http://127.0.0.1:10010 |
| Gateway düğümü 2 | http://127.0.0.1:10011 |
| HTTPS, düğüm 1 | https://127.0.0.1:18443 |
| HTTP/2, düğüm 1 | http://127.0.0.1:10012 |
| WebSocket, düğüm 1 | http://127.0.0.1:10013 |
| gRPC, düğüm 1 | 127.0.0.1:10014 |
| SSE, düğüm 1 | http://127.0.0.1:10015 |
| TCP, düğüm 1 | 127.0.0.1:10020 |
| UDP, düğüm 1 | 127.0.0.1:10021 |
| TLS geçişi, düğüm 1 | 127.0.0.1:18444 |

`127.0.0.1:10000` iki Envoy düğümünün önündeki yerel dağıtıcıdır. Bir düğüm durunca yeni bağlantı diğerine gider. Bu, rack veya veri merkezi HA kanıtı değildir.

PostgreSQL hosta yayınlanmaz. Konteyner içi bağlantı `postgres:5432`, kullanıcı `gateway`, parola `gateway-dev` yalnızca yerel Compose içindir.

Veritabanı parolası yalnızca yerel Compose içindir. Üretim gizli yönetimi sayılmaz. Envoy admin, metrik ve xDS portları hosta açılmaz.

## Örnek rota

Ana sayfada "Yeni bir site veya API'yi yayına al" seçin ve sihirbazı izleyin:

1. Adres: `demo.local`, sorumlu: herhangi bir ekip adı
2. Hedef: `backend-a`, port `8080`, yol `/a`
3. Kontrol: taslak kaydedilir, canlı trafik değişmez
4. Yayına al: tüm sunucular veya önce tek sunucu
5. Dene: sunucu durumu ve kopyalanabilir komut

```sh
curl -sS -H 'Host: demo.local' http://127.0.0.1:10000/a
```

Yanıt `instance=backend-a` olmalıdır. Bilinmeyen hostname 404 döner; tarayıcıya yalnızca `127.0.0.1:10000` yazmak bu yüzden 404 verir. Sihirbaz servis başına bir yol kurar. Ek yollar API ile eklenir ve sihirbaz onları korur. `make verify` iki yollu örneği API üzerinden kurar.

Otomatik kontrol:

```sh
make verify
```

## Durdurma

```sh
make down
```

Kimlikler `.local/` altındadır ve git'e girmez. Silinirse `make bootstrap` yenilerini üretir; eski imzalı paketler doğrulanamaz.
