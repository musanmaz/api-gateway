# Kapasite

Trafik hacmi, eşzamanlı bağlantı, yanıt boyutu ve donanım bilinmiyor. Bu yüzden RPS, TLS el sıkışması veya NIC doygunluğu için sayı verilmez.

Ölçülmeden önce hedef makinede CPU, bellek, NIC, disk, dosya tanıtıcısı ve conntrack tüketimi, aynı koşullardaki doğrudan upstream ölçümüyle karşılaştırılmalıdır. Compose üzerinde konteyner durdurmak rack kaybı kanıtı değildir.

Bu sürümde devre kesici bağlantı ve istek tavanı 1024, retry bütçesi 0'dır. Bu bir kapasite ölçümü değildir.
