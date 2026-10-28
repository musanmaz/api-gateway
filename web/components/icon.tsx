import type { SVGProps } from "react";
const paths = {
  grid: "M3 3h7v7H3z M14 3h7v7h-7z M3 14h7v7H3z M14 14h7v7h-7z",
  layers: "m12 3 9 5-9 5-9-5 9-5Z M3 12l9 5 9-5 M3 16l9 5 9-5",
  nodes: "M4 3h16v6H4z M4 15h16v6H4z M8 6h.01 M8 18h.01 M12 9v6",
  history: "M3 4v5h5 M3 9a9 9 0 1 1 1 9 M12 7v5l3 2",
  shield: "m12 3 8 3v6c0 5-8 9-8 9s-8-4-8-9V6l8-3Z m-4 9 3 3 5-6",
  lab: "M9 3h6 M10 3v7L4 20a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1l-6-10V3 M7 15h10",
  arrow: "M4 12h16 m-6-6 6 6-6 6",
  chevron: "m9 5 7 7-7 7",
  plus: "M12 5v14 M5 12h14",
  search: "M21 21l-5-5 M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0",
  logout: "M9 4H4v16h5 M12 12h9 m-4-4 4 4-4 4",
  refresh: "M20 7v5h-5 M4 17v-5h5 M5.5 7a7.5 7.5 0 0 1 12.8-2L20 8 M4 16l1.7 3a7.5 7.5 0 0 0 12.8-2",
  close: "m6 6 12 12 M6 18 18 6",
  menu: "M4 6h16 M4 12h16 M4 18h16",
  terminal: "m5 7 5 5-5 5 M13 17h6",
  check: "m5 12 4 4L19 6",
  external: "M14 3h7v7 M21 3 10 14 M10 3H3v18h18v-7",
  clock: "M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0 M12 7v5l3 2",
  activity: "M2 12h4l3-8 6 16 3-8h4",
  globe: "M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0 M3 12h18 M12 3a18 18 0 0 1 0 18 18 18 0 0 1 0-18",
  copy: "M9 9h12v12H9z M15 9V3H3v12h6",
  lock: "M5 10h14v11H5z M8 10V6a4 4 0 0 1 8 0v4 M12 14v3",
  info: "M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0 M12 11v6 M12 7h.01",
  user: "M16 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0 M4 21v-3a8 8 0 0 1 16 0v3",
};
export type IconName = keyof typeof paths;
export function Icon({ name, size = 18, ...props }: SVGProps<SVGSVGElement> & { name: IconName; size?: number }) {
  return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.65" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" {...props}><path d={paths[name]} /></svg>;
}
export function BrandMark() {
  return <span className="brand-mark"><Icon name="layers" size={24} /></span>;
}
