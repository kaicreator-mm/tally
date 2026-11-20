import { StrictMode, useEffect, useState } from 'react';
import { createRoot } from 'react-dom/client';
import '../styles/tokens.css';
import '../styles/app.css';
import { App } from './App.tsx';
import { createRouter, type Route } from './router.ts';
import { ShellProvider } from './shell/AppContext.tsx';
import { detectLocale, rememberLocale } from '../i18n/index.ts';
import { orNotConnected, hostAdapter } from '../adapters/host.ts';
import type { Catalog } from '../adapters/host.ts';

const router = createRouter({
  getHash: () => window.location.hash || '#/workbench/inbox',
  setHash: (hash) => { window.location.hash = hash; },
  addEventListener: (type, fn) => window.addEventListener(type, fn),
  removeEventListener: (type, fn) => window.removeEventListener(type, fn),
});
const locale = detectLocale({ storage: window.localStorage });
rememberLocale(window.localStorage, locale);

function Root() {
  const [route, setRoute] = useState<Route>(() => router.get());
  // UI-E: project/version catalog loads from the Host read model; an outage
  // keeps the shell usable with an empty (NOT_CONNECTED) catalog.
  const [catalog, setCatalog] = useState<Catalog>({connected:false});
  useEffect(() => {
    void orNotConnected('catalog', () => hostAdapter.catalog())
      .then(result => setCatalog(result as Catalog));
  }, []);
  useEffect(() => {
    const off = router.subscribe(next => {
      setRoute(next);
      document.getElementById('main')?.focus({ preventScroll: true });
    });
    return off;
  }, []);
  return (
    <ShellProvider storage={window.localStorage}
      catalog={catalog.connected?{
        projects:catalog.projects??[],
        versionsByProject:catalog.versionsByProject??{},
      }:{projects:null,versionsByProject:null}}>
      <App route={route} onNavigate={hash => router.navigate(hash)} />
    </ShellProvider>
  );
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Root />
  </StrictMode>,
);
