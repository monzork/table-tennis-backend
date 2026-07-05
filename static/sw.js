const CACHE_NAME = 'open-tdm-v1';
const ASSETS_TO_CACHE = [
  '/static/dist/tailwind.css',
  '/static/fonts/outfit.css',
  '/open_tdm.jpeg',
  '/favicon.ico',
  '/static/manifest.json'
];

self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => cache.addAll(ASSETS_TO_CACHE))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(keys => {
      return Promise.all(
        keys.filter(key => key !== CACHE_NAME)
            .map(key => caches.delete(key))
      );
    })
  );
  self.clients.claim();
});

self.addEventListener('fetch', event => {
  // Only cache GET requests to our origin
  if (event.request.method !== 'GET' || !event.request.url.startsWith(self.location.origin)) {
    return;
  }
  
  // For static assets, do a cache-first approach
  if (event.request.url.includes('/static/') || event.request.url.includes('.jpeg') || event.request.url.includes('.ico')) {
    event.respondWith(
      caches.match(event.request).then(cachedResponse => {
        return cachedResponse || fetch(event.request).then(response => {
          return caches.open(CACHE_NAME).then(cache => {
            cache.put(event.request, response.clone());
            return response;
          });
        });
      })
    );
    return;
  }

  // For HTML pages (HTML/HTMX requests), do a network-first approach
  event.respondWith(
    fetch(event.request)
      .then(response => {
        // Optionally cache the HTML pages if you want offline support
        return response;
      })
      .catch(() => {
        return caches.match(event.request);
      })
  );
});
