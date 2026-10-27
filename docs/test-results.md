# Test sonuçları

Tarih: 2026-10-26T07:46:30Z
Ortam: Docker Compose, tek makine, iki gateway düğümü ve yerel edge. Bu sonuç rack, veri merkezi veya üretim HA kanıtı değildir.

| Test | Sonuç | Not |
| --- | --- | --- |
| Arayüz/API servisi iki backend'e gider | PASS | `e2e-1793000746.local` için `/a` backend-a, POST `/b` backend-b. Bilinmeyen host 404. İki düğüm de yanıtladı. |
| Viewer yayım yapamaz | PASS | HTTP 403. |
| Aynı idempotency anahtarı ikinci yayım açmaz | PASS | Revision 23 tekrarlandı. |
| Yayım farkı | PASS | Taslakta `/a` eklenecek olarak göründü. |
| Elle geri alma | PASS | Revision 23 paketinden revision 28 üretildi. İkinci hostname kalktı. Güncel pakete tekrar dönüş 409. Viewer 403. |
| Yönetim API durunca mevcut trafik sürer | PASS | API konteyneri durduruldu. |
| Merkez kapalıyken iki düğüm yeniden başlar | PASS | Doğrulanmış diskten iki düğüm ve edge trafik verdi. |
| Bozuk yerel önbellek ve merkez kapalı | PASS | İki düğümün dosyaları bozulunca HTTP 200 alınmadı. API dönünce trafik toparlandı. |
| Bir düğüm durunca yeni bağlantı | PASS | gw-local-1 durdurulunca edge 724 ms içinde gw-local-2 üzerinden yanıt verdi. Ortak üretim VIP'i yok. |
| İmza ve çakışma birim testleri | PASS | `go test ./...` |
| TLS, yanlış SAN, sertifika yenileme | NOT RUN | Public TLS yok. Sağlayıcı bilinmiyor. |
| NACK, canary, eşzamanlı yazıcı | PASS | Aynı expectedVersion ve iki idempotency anahtarı 200 ve 409 verdi. Canary revision 25 yalnız gw-local-1. NACK revision 26 ACK'i 25 bıraktı; disk current bu revision olmadı. Edge kanıtı değildir. |
| WebSocket, SSE, gRPC, büyük yükleme | PASS | 2026-10-23 düğüm 1 ölçümü. Bu koşu protokolleri yeniden ölçmedi. Büyük yükleme ayrıca NOT RUN. |
| CSRF, sahte istemci IP, SSRF uçtan uca | NOT RUN | Loopback ve metadata adresi birim testinde reddedilir. |
| OIDC, kota ve WAF | NOT RUN | Kimlik sağlayıcı ve bu bağımlılıklar yok. |
| Telemetri diski dolu | NOT RUN | Compose log sınırı 10m x 3 tanımlı, doluluk deneyi yapılmadı. |
| PostgreSQL primary ve quorum | NOT RUN | Tek PostgreSQL var. |
| Yük testi ve eklenen gecikme | NOT RUN | Ölçüm yok. |
| Yedekten geri dönüş | NOT RUN | Yedek alınmadı. |
| Lokasyon kaybı | NOT RUN | İkinci lokasyon yok. |

## Protokoller, 2026-10-23

Düğüm 1 üzerinde, yayım 16 ve sonrası. Host portları `127.0.0.1` üzerindedir. Bu tablo bu koşuda yeniden ölçülmedi.

| Protokol | Sonuç | Not |
| --- | --- | --- |
| HTTPS | PASS | `https-1792682249.test.local:18443`, yerel CA, HTTP/2, `instance=backend-a`. |
| HTTP/2 | PASS | Port 10012, prior knowledge. |
| WebSocket | PASS | Port 10013, 101 ve `instance=backend-a`. |
| SSE | PASS | Port 10015, `data: instance=backend-a`. |
| gRPC | PASS | Port 10014, health `serving`. |
| TCP | PASS | Port 10020, `/healthz` gövdesi `ok`. |
| UDP | PASS | `envoy:10021` docker ağında `pong`. Hosttaki `127.0.0.1:10021` bu Docker Desktop oturumunda yanıt vermedi. |
| TLS geçişi | PASS | Port 18444 sertifikası backend sertifikasıyla aynı. Gövde `passthrough instance=backend-a`. |

