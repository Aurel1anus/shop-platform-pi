import type { ExtensionAPI } from '@earendil-works/pi-coding-agent';
import { registerCatalogTools } from './catalog.ts';

export default function commerce(pi: ExtensionAPI) {
  registerCatalogTools(pi);
}
