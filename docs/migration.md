# Geçiş

Nginx veya OpenResty içe aktarma bu sürümde yoktur. Yetkisiz ağ taraması yapılmaz.

Sonraki aşamada kullanıcı tarafından verilen konfigürasyon salt okunur incelenir. `server_name`, port, upstream, `proxy_pass`, timeout, header, TLS referansı ve ACL ayrıştırılmaya çalışılır. Secret değerleri rapordan çıkarılır. Lua, map, belirsiz regex, auth, cache ve dinamik değişkenler sessizce dönüştürülmez; uyumsuzluk raporuna yazılır.

Geçiş sırası: mevcut vekili upstream yapmak, Host/SNI koruyan test, düşük trafik, gözlem, yaygınlaştırma. Uygulama sağlığı uygun değilse geçiş büyütülmez. DNS ve yük dengeleyici değişimi, TTL beklemesi ve eski vekile dönüş servis bazında yazılır. Origin'e doğrudan erişimi kapatmak bir plandır; bu depo onu üretimde uygulamaz.
