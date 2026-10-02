// Service worker do App do Membro (PWA).
//
// Estrategia deliberadamente conservadora por causa do multi-tenant:
// - NUNCA cacheia /api/* (dados pessoais/sensiveis de cada igreja).
// - Assets do Next (_next/static, icones, fontes) sao imutaveis -> cache-first.
// - Navegacao -> network-first; se offline, cai na tela /offline.
//
// Ao mudar a estrategia, suba a versao de CACHE para invalidar o cache antigo.

const CACHE = "chosen-member-v2";
const OFFLINE_URL = "/offline";
const PRECACHE = [
  OFFLINE_URL,
  "/icon.svg",
  "/icon-192.png",
  "/icon-512.png",
  "/maskable-512.png",
  "/apple-touch-icon.png",
];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches
      .open(CACHE)
      .then((cache) => cache.addAll(PRECACHE))
      .then(() => self.skipWaiting()),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((key) => key !== CACHE).map((key) => caches.delete(key))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") return;

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;
  // Dados sempre frescos e fora do cache (privacidade).
  if (url.pathname.startsWith("/api/")) return;

  if (request.mode === "navigate") {
    event.respondWith(fetch(request).catch(() => caches.match(OFFLINE_URL)));
    return;
  }

  if (url.pathname.startsWith("/_next/static/") || /\.(?:png|svg|ico|webp|woff2?)$/.test(url.pathname)) {
    event.respondWith(
      caches.match(request).then(
        (hit) =>
          hit ||
          fetch(request).then((res) => {
            if (res.ok) {
              const copy = res.clone();
              caches.open(CACHE).then((cache) => cache.put(request, copy));
            }
            return res;
          }),
      ),
    );
  }
});

// Web Push: exibe a notificacao recebida do servidor (payload JSON).
self.addEventListener("push", (event) => {
  let data = { title: "App do Membro", body: "Você tem uma novidade." };
  if (event.data) {
    try {
      data = { ...data, ...event.data.json() };
    } catch {
      data.body = event.data.text();
    }
  }
  event.waitUntil(
    self.registration.showNotification(data.title || "App do Membro", {
      body: data.body || "",
      icon: "/icon-192.png",
      badge: "/icon-192.png",
      data: { url: data.url || "/home" },
    }),
  );
});

// Ao tocar na notificacao, foca a janela aberta ou abre a URL indicada.
self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const url = (event.notification.data && event.notification.data.url) || "/home";
  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((list) => {
      for (const client of list) {
        if ("focus" in client) {
          client.navigate(url);
          return client.focus();
        }
      }
      return self.clients.openWindow(url);
    }),
  );
});

