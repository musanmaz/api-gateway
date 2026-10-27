# Yedeklilik ve kurtarma

Buradaki sayılar üretim referansıdır. Bu depodaki Compose bunları çalıştırmaz ve ölçülmüş RPO/RTO sunmaz.

## Referans yerleşim

Public giriş, mevcut yedekli L4 yük dengeleyici olmalıdır. Yoksa aynı L2 ağda iki HAProxy/Keepalived düğümü düşünülür. VIP sahipliği active/passive olabilir. VIP tek arıza noktası gibi bırakılmaz. VRRP aynı L2 veya bilinen unicast komşuları içinde kalır. Farklı veri merkezleri arasına uzatılmış VIP, bölünmüş beyin ve ARP/NDP güncellemesi yüzünden HA sayılmaz.

Her filoda en az iki Envoy ve agent, farklı host ve mümkünse farklı rack üzerinde durur. Biri gidince kalan kapasite yeni bağlantıları alır. Aynı hypervisor, depolama, güç veya switch üzerindeki kopya bağımsız yedek değildir.

Yönetim API'si en az iki instance olur ve oturum süreç belleğine bağlanmaz. Yayım işleri birden fazla worker ile çalışır; her işin tek yürütücüsü lease ve fencing ile seçilir. Bu kod bugün tek API sürecidir.

PostgreSQL için referans Patroni ve üç veri düğümüdür. Kendi consensus algoritması yazılmaz. Senkron replikasyon yazma sırasında bir replika bekler; o replika gidince yazma durabilir, veri kaybı penceresi daralır. Asenkron replikasyon yazmayı sürdürür ve failover'da son işlemler kaybolabilir. RPO=0 yalnızca senkron mod ve o senaryo için söylenebilir. Quorum kaybında iki primary üretilmez. Eski primary fencing edilmeden geri alınmaz. Quorum yokken yeni konfigürasyon yayımı durur. Çalışan Envoy, elindeki son geçerli paketle trafiğe devam eder.

İki lokasyona 2+1 quorum koymak, her iki lokasyonun kaybına eşit dayanıklılık değildir. Üçüncü bağımsız oy alanı yoksa bir lokasyonun kaybı yazmayı durdurabilir. İkinci veri merkezinde ayrı data plane ve yerel backend sağlığı gerekir. GSLB veya DNS failover, uygulama verisinin durumuna ve backend kapasitesine bağlıdır. DNS TTL, failover süresi değildir. Mevcut bağlantıların kesilmeyeceği vaat edilmez. Geri dönüş kontrollü ve çoğu zaman manueldir.

Gateway yedekliliği, uygulama veritabanını veya kullanıcı oturumunu yedeklemez.

## Ortak arıza

Birlikte konan API ve veritabanı, disk veya host kaybında birlikte gider. Telemetri ile trafik aynı disk dolunca birlikte etkilenebilir; bu yüzden gözlem ayrı kaynakta durmalıdır. Telemetri arızası gateway readiness'ini kapatmamalıdır. Readiness, geçerli güvenlik materyali ve gerçekten trafik alabilmektir.

Bağımlılıklar: DNS, resolver, NTP, firewall/NAT, L4 LB, IdP, gizli alan ve yönetim ağı. Bunların her biri ayrıca arızalanabilir.

## Hedefler, ölçüm değil

Tasarım hedefi: data-plane için %99,95, tek trafik düğümü kaybında yeni bağlantıların 10 saniye içinde toparlanması, merkez kaybında son geçerli konfigürasyonla çalışma, yönetim verisi için RPO 5 dakika ve RTO 60 dakika. Tek makinede bir düğüm durdurma süresi `docs/test-results.md` içindedir. Bu, üretim VIP'i veya veri merkezi ölçümü değildir. RPO ve RTO bu ortamda ölçülmedi.

Graceful drain, bağlantı boşaltma, overload ve N+1 hesabı sonraki kapasite çalışmasındadır. Tüm upstream'ler sağlıksızken hassas servise otomatik fail-open yapılmayacak.
