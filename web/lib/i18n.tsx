"use client";

import {createContext,useCallback,useContext,useEffect,useMemo,useState,ReactNode} from "react";

export type Language="tr"|"en";
const english:Record<string,string>={
  "Genel bakış":"Overview", "Servisler":"Services", "Gateway düğümleri":"Gateway nodes",
  "Test laboratuvarı":"Test lab", "Yayın geçmişi":"Deployments",
  "Sertifikalar":"Certificates", "İşlem kayıtları":"Audit logs",
  "Çalışma alanı":"Workspace", "Yönetim":"Management", "Ana menü":"Main navigation",
  "Kenar menü":"Sidebar", "İçeriğe geç":"Skip to content", "Menüyü kapat":"Close menu",
  "Menüyü aç":"Open menu", "Saat dilimi":"Time zone","Dil":"Language",
  "Test ortamı":"Test environment", "Yönetim oturumu":"Admin session",
  "Oturumu kapat":"Sign out", "Servis operasyonları":"Service operations",
  "Kontrol sizde. Trafik Envoy’da.":"Your controls. Powered by Envoy.",
  "Bağlı":"Connected", "Güncel değil":"Stale", "Bilinmiyor":"Unknown",
  "Taslak":"Draft", "Yayımlandı":"Published",
  "Yeni servis":"New service", "Yeni site veya API'yi yayına al":"Publish a new site or API",
  "Adres":"Address", "Hedef":"Target", "Kontrol":"Review", "Yayına al":"Publish",
  "Dene":"Test", "Geri":"Back", "Devam":"Continue",
  "Gelişmiş ayarlar":"Advanced settings", "Gelişmiş ayarları gizle":"Hide advanced settings",
  "Servis kısa adı":"Service slug", "Ek özel alan adı (isteğe bağlı)":"Optional custom hostname",
  "Bu servisten kim sorumlu?":"Service owner",
  "Hedef sunucu":"Upstream server", "Port":"Port", "Hangi yol?":"Path",
  "Bağlantı türü":"Connection type", "Yol eşleşmesi":"Path matching",
  "Bu yolla başlayan her şey":"All paths beginning with this prefix",
  "Yalnızca tam bu yol":"Only this exact path",
  "Backend kimlik doğrulaması ve header'lar":"Upstream authentication & headers",
  "Backend auth türü":"Upstream auth type", "Backend kullanıcı adı":"Upstream username",
  "Backend parolası":"Upstream password", "Bearer token":"Bearer token",
  "Header adı":"Header name", "Header değeri":"Header value",
  "İstemciden zorunlu header (isteğe bağlı)":"Required request header (optional)",
  "Rate limiting":"Rate limiting", "Rate limit kaydet":"Save rate limit",
  "Servis güvenliği":"Service security",
  "API Key zorunlu":"API Key required",
  "Public — kimlik doğrulaması yok":"Public — no authentication",
  "API Key oluştur ve korumayı aç":"Generate API Key & enable protection",
  "API Key yenile":"Rotate API Key","Auth kapat (Public)":"Disable authentication (Public)",
  "Tüm sunuculara gönder":"Publish to all nodes",
  "Yalnızca bu sunucuya gönder":"Publish to this node only",
  "Servis rotaları":"Service routes", "Alan adından hedef servise":"Domains to upstream services",
  "Tüm servisler":"All services", "Yayımlanan":"Published",
  "Servis ara":"Search services", "İstemci":"Client",
  "HTTPS girişi":"HTTPS ingress", "Backend":"Backend",
  "Gateway genel bakış":"Gateway overview",
  "Operasyon merkezi":"Operations center",
  "Yayındaki servisler":"Active services","Bağlı düğümler":"Online nodes",
  "Aktif revizyon":"Active revision","Bekleyen taslaklar":"Pending drafts",
  "Yönlendirmeyi gerçek isteklerle doğrula.":"Verify routing with real requests.",
  "Test ortamını aç":"Open test lab","Son yayınlar":"Recent deployments",
  "Ayrıntıları gör":"View details",
  "Tekrar hoş geldiniz.":"Welcome back.","Giriş yap":"Sign in", "Servislerinizi yönetmek için erişim anahtarınızı girin.":"Enter your access key to manage services.",
  "Gateway Yönetim Paneli":"Gateway Admin Console",
  "Erişim anahtarı":"Access key", "Erişim anahtarınız":"Your access key",
  "Doğrulanıyor…":"Verifying…",
  "Servis / Gateway erişim linki":"Service / Gateway URL",
  "Yayın durumu":"Deploy status","Sorumlu":"Owner",
  "Özel alan adı yok":"No custom domain",
  "HTTP gateway linki yok":"No HTTP gateway URL",
  "Yayın sonrası":"After publishing",
  "Servislerden biri bulunamadı":"Service not found",
  "Henüz gösterilecek servis yok.":"No services to display.",
  "İlk servisinizi ekleyin":"Add your first service",
  "Yeni API'leriniz için endpoint'i, güvenliği ve politikaları tanımlayın.":"Configure endpoints, security and policies for your new APIs.",
  "İstek/saniye":"Requests/second",
  "Endpoint bazlı":"Per endpoint",
  "Endpoint politikalarını karşılaştır":"Compare endpoint policies",
  "Endpoint":"Endpoint",
  "Kayıtlı kota":"Saved limit",
  "0 olan endpoint ayarı servis kotasını miras alır. Burada etkin kotalar gösterilir.":"An endpoint limit of 0 inherits the service limit. Effective quotas are shown here.",
  "Güvenlik politikası":"Security policy",
  "Kayıtlı güvenlik politikaları":"Saved security policies",
  "Engelle":"Block",
  "İzle":"Detect",
  "Kapalı":"Off",
  "Rate limit açık":"Rate limit enabled",
  "Kota yok":"No quota",
  "Public":"Public",
  "Yayın bekliyor":"Pending publish",
  "istek/sn/düğüm":"req/s/node",
  "Limitsiz":"Unlimited",
  "Güvenlik ve Trafik":"Security & Traffic",
  "Kaydedilen politikalar ve düğümlerde uygulanan gerçek değerler.":"Saved policies and the actual configurations running on each node.",
  "Politikaları düzenle":"Edit policies",
  "Yayın bekleyen değişiklikler var.":"There are unpublished changes.",
  "Kaydedilen politika henüz canlı sürümle aynı değil. Düğüm tablosunu kontrol edin.":"The saved policy is not yet the live version. Check the node table.",
  "Servis genelinde WAF politikası.":"Service-level WAF policy.",
  "Endpoint bazlı istisnalar var.":"Some endpoints have their own limits.",
  "Düğüm başına; global kota değil.":"Per node; not a global distributed quota.",
  "İstemci erişimi":"Client access",
  "Allowlist":"Allowlist",
  "Denylist":"Denylist",
  "Backend kimlik doğrulaması":"Upstream authentication",
  "Zorunlu header":"Required header",
  "Ek istemci header koşulu yok.":"No additional required client header.",
  "Düğüm bazında uygulanan politika":"Policy applied per node",
  "Envoy ACK ile doğrulanan revizyondan okunur; bağlantısı olmayan düğüm aktif sayılmaz.":"Read from the Envoy-acknowledged revision; disconnected nodes are not considered active.",
  "Politika durumu yükleniyor…":"Loading policy state…",
  "Politika doğrulanamadı.":"Could not verify the policy.",
  "Düğüm":"Node",
  "Durum":"Status",
  "Erişim":"Access",
  "Revizyon":"Revision",
  "ACK onaylı":"ACK confirmed",
  "Çevrimdışı":"Offline",
  "Eski sürüm":"Outdated",
  "Kurulmadı":"Not deployed",
  "İstenen":"Desired",
  "HTTP ve WAF olay sayaçları şu anda düğüm toplamıdır, bu servise özel değildir.":"HTTP and WAF event counters currently aggregate the whole node; they are not service-specific.",
  "Düğüm metriklerini aç":"Open node metrics",
};
type LanguageContextValue={
  language:Language;
  setLanguage:(lang:Language)=>void;
  t:(key:string)=>string;
};
const LanguageContext=createContext<LanguageContextValue>({
  language:"tr",setLanguage:()=>{},t:k=>k
});
export function LanguageProvider({children}:{children:ReactNode}){
 const [language,setLanguage]=useState<Language>("tr");
 useEffect(()=>{
   const stored=window.localStorage.getItem("gateway.language");
   if(stored==="en"||stored==="tr")setLanguage(stored);
 },[]);
 useEffect(()=>{
   document.documentElement.lang=language;
   window.localStorage.setItem("gateway.language",language);
 },[language]);
 const t=useCallback((key:string)=>language==="en"?(english[key]??key):key,[language]);
 const value=useMemo(()=>({language,setLanguage,t}),[language,t]);
 return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>;
}
export function useLanguage(){return useContext(LanguageContext);}
