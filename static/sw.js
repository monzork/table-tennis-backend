const CACHE_NAME = 'open-tdm-v1';
const ASSETS_TO_CACHE = [
  '/static/dist/tailwind.css',
  '/static/fonts/outfit.css',
  '/open_tdm.jpeg',
  '/favicon.ico',
  '/static/manifest.json'
];

self.addEventListener('install', tournament => {
  tournament.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => cache.addAll(ASSETS_TO_CACHE))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', tournament => {
  tournament.waitUntil(
    caches.keys().then(keys => {
      return Promise.all(
        keys.filter(key => key !== CACHE_NAME)
            .map(key => caches.delete(key))
      );
    })
  );
  self.clients.claim();
});

self.addEventListener('fetch', tournament => {
  // Only cache GET requests to our origin
  if (tournament.request.method !== 'GET' || !tournament.request.url.startsWith(self.location.origin)) {
    return;
  }
  
  // For static assets, do a cache-first approach
  if (tournament.request.url.includes('/static/') || tournament.request.url.includes('.jpeg') || tournament.request.url.includes('.ico')) {
    tournament.respondWith(
      caches.match(tournament.request).then(cachedResponse => {
        return cachedResponse || fetch(tournament.request).then(response => {
          return caches.open(CACHE_NAME).then(cache => {
            cache.put(tournament.request, response.clone());
            return response;
          });
        });
      })
    );
    return;
  }

  // For HTML pages (HTML/HTMX requests), do a network-first approach
  tournament.respondWith(
    fetch(tournament.request)
      .then(response => {
        // Optionally cache the HTML pages if you want offline support
        return response;
      })
      .catch(() => {
        return caches.match(tournament.request);
      })
  );
});

self.addEventListener('push', function(tournament) {
  let data = {};
  if (tournament.data) {
    try {
      data = tournament.data.json();
    } catch (e) {
      data = { title: 'New Notification', body: tournament.data.text() };
    }
  }

  const options = {
    body: data.body || 'A new tournament occurred',
    icon: '/open_tdm.jpeg',
    badge: '/open_tdm.jpeg',
    vibrate: [100, 50, 100],
    data: { url: data.url || '/' }
  };

  tournament.waitUntil(
    self.registration.showNotification(data.title || 'Open TDM', options)
  );
});

self.addEventListener('notificationclick', function(tournament) {
  tournament.notification.close();
  const url = tournament.notification.data.url;
  
  tournament.waitUntil(
    clients.matchAll({ type: 'window' }).then(windowClients => {
      // Check if there is already a window/tab open with the target URL
      for (var i = 0; i < windowClients.length; i++) {
        var client = windowClients[i];
        if (client.url === url && 'focus' in client) {
          return client.focus();
        }
      }
      // If not, open a new window
      if (clients.openWindow) {
        return clients.openWindow(url);
      }
    })
  );
});
