import './style.css';
import { renderHome } from './pages/home';

// Wiring convention: src/api owns transport, src/pages own screens,
// src/components own reusable UI, src/lib owns shared utilities.
renderHome(document.querySelector<HTMLDivElement>('#app')!);
