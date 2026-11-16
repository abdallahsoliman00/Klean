/* Every icon in the app. Line icons use currentColor, so they follow the text colour. */
import type { SVGProps } from 'react';

type IconProps = { size?: number; className?: string };

/** A line icon on a 16-unit grid. */
function Line({ size = 14, className, children, ...rest }: IconProps & SVGProps<SVGSVGElement>) {
  return (
    <svg className={className} width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth={1.5} {...rest}>
      {children}
    </svg>
  );
}

export const CopyIcon = (p: IconProps) => (
  <Line {...p}><rect x="5" y="5" width="8.5" height="8.5" rx="2" /><path d="M10.5 5V3.8A1.3 1.3 0 009.2 2.5H3.8a1.3 1.3 0 00-1.3 1.3v5.4a1.3 1.3 0 001.3 1.3H5" /></Line>
);
export const CheckIcon = (p: IconProps) => (
  <Line {...p} strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><path d="M3 8.5l3 3 7-7" /></Line>
);
export const RedoIcon = (p: IconProps) => (
  <Line {...p} strokeLinecap="round" strokeLinejoin="round"><path d="M13 3v3.5H9.5" /><path d="M12.6 6.5A5 5 0 103 9" /></Line>
);
export const EditIcon = (p: IconProps) => (
  <Line {...p} strokeLinejoin="round"><path d="M10.5 2.5l3 3L6 13H3v-3z" /></Line>
);
export const DotsIcon = ({ size = 14 }: IconProps) => (
  <svg width={size} height={size} viewBox="0 0 16 16" fill="currentColor"><circle cx="3.5" cy="8" r="1.3" /><circle cx="8" cy="8" r="1.3" /><circle cx="12.5" cy="8" r="1.3" /></svg>
);
export const TrashIcon = (p: IconProps) => (
  <Line {...p} strokeLinecap="round"><path d="M3 4.5h10M6.5 4.5V3h3v1.5M4.5 4.5l.6 8.5h5.8l.6-8.5" /></Line>
);
export const StarIcon = ({ filled, ...p }: IconProps & { filled?: boolean }) => (
  <Line {...p} strokeLinejoin="round" fill={filled ? 'currentColor' : 'none'}><path d="M8 2.2l1.75 3.7 4 .5-2.95 2.8.75 4L8 11.25 4.45 13.2l.75-4L2.25 6.4l4-.5z" /></Line>
);
export const PlusIcon = (p: IconProps) => (
  <Line {...p} strokeWidth={1.6} strokeLinecap="round"><path d="M8 3v10M3 8h10" /></Line>
);
export const SearchIcon = (p: IconProps) => (
  <Line {...p}><circle cx="7" cy="7" r="4.5" /><path d="M10.5 10.5L14 14" strokeLinecap="round" /></Line>
);
export const PageIcon = (p: IconProps) => (
  <Line {...p} strokeLinecap="round" strokeLinejoin="round"><path d="M4 2.5h5.5L12 5v8.5H4z" /><path d="M6.2 7.5h3.6M6.2 10.2h3.6" /></Line>
);
export const AlertIcon = (p: IconProps) => (
  <Line {...p} strokeLinecap="round"><circle cx="8" cy="8" r="5.8" /><path d="M8 5v3.4M8 10.9v.1" /></Line>
);
export const AskIcon = (p: IconProps) => (
  <Line {...p} strokeLinecap="round" strokeLinejoin="round"><path d="M6.1 6.2a2 2 0 113 1.7c-.7.4-1.1.9-1.1 1.6M8 11.6v.1" /><circle cx="8" cy="8" r="5.8" /></Line>
);
export const ChevronIcon = () => (
  <svg className="chev" width="10" height="10" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round"><path d="M4.5 3l3 3-3 3" /></svg>
);
export const SpinnerIcon = (p: IconProps) => (
  <Line {...p} className="spin" strokeWidth={1.6} strokeLinecap="round"><path d="M8 2.2a5.8 5.8 0 105.8 5.8" /></Line>
);
export const DownloadIcon = (p: IconProps) => (
  <Line size={16} {...p} strokeLinecap="round" strokeLinejoin="round"><path d="M8 2.5v7.5M4.8 7.2L8 10.3l3.2-3.1M3 13h10" /></Line>
);
/** The tick inside a radio or checkbox option. */
export const ExpandIcon = (p: IconProps) => (
  <Line {...p} strokeLinecap="round" strokeLinejoin="round"><path d="M9.5 2.5h4v4M13.5 2.5L9 7M6.5 13.5h-4v-4M2.5 13.5L7 9" /></Line>
);
export const ShrinkIcon = (p: IconProps) => (
  <Line {...p} strokeLinecap="round" strokeLinejoin="round"><path d="M13.5 6.5h-4v-4M9.5 6.5l4-4M2.5 9.5h4v4M6.5 9.5l-4 4" /></Line>
);
export const MarkIcon = () => (
  <Line size={11} strokeWidth={2.4} strokeLinecap="round" strokeLinejoin="round"><path d="M3.5 8.5l3 3 6-6.5" /></Line>
);
export const RefreshIcon = ({ spinning }: { spinning?: boolean }) => (
  <Line size={12} className={spinning ? 'spin' : undefined} strokeWidth={1.6} strokeLinecap="round" strokeLinejoin="round"><path d="M13 3v3.5H9.5" /><path d="M12.6 6.5A5 5 0 103 9" /></Line>
);
export const CloseIcon = () => (
  <svg width="16" height="16" viewBox="0 0 16 16" stroke="currentColor" strokeWidth={1.6} strokeLinecap="round"><path d="M4 4l8 8M12 4l-8 8" /></svg>
);
export const CaretDownIcon = () => (
  <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth={1.6}><path d="M3 4.5l3 3 3-3" /></svg>
);
export const GearIcon = () => (
  <Line size={16}><circle cx="8" cy="8" r="2.2" /><path d="M8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.4 3.4l1.4 1.4M11.2 11.2l1.4 1.4M3.4 12.6l1.4-1.4M11.2 4.8l1.4-1.4" strokeLinecap="round" /></Line>
);
export const MoonIcon = () => (
  <Line size={16}><path d="M13.5 9.5A5.5 5.5 0 016.5 2.5a5.5 5.5 0 107 7z" strokeLinejoin="round" /></Line>
);
export const SidebarIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth={1.5}><rect x="2.5" y="3" width="13" height="12" rx="2.5" /><path d="M7 3v12" /></svg>
);
export const PanelIcon = () => (
  <Line size={16} strokeLinejoin="round"><rect x="2.5" y="2.5" width="11" height="11" rx="2.5" /><path d="M9.5 2.5v11" /></Line>
);
export const SlidersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round"><path d="M3 5h7M13 5h2M3 13h2M8 13h7" /><circle cx="11.5" cy="5" r="1.5" /><circle cx="6.5" cy="13" r="1.5" /></svg>
);
export const LinesIcon = () => (
  <Line><path d="M3 3.5h10M3 8h10M3 12.5h6" strokeLinecap="round" /></Line>
);
export const WrenchIcon = () => (
  <Line strokeLinecap="round" strokeLinejoin="round"><path d="M9.6 2.6a3.4 3.4 0 00-3.3 4.5L2.7 10.7a1.5 1.5 0 002.1 2.1l3.6-3.6a3.4 3.4 0 004.4-4.3l-2.1 2.1-1.9-.5-.5-1.9z" /></Line>
);
export const ImageIcon = (p: IconProps) => (
  <Line {...p} strokeLinecap="round" strokeLinejoin="round"><rect x="2.5" y="3" width="11" height="10" rx="2.2" /><circle cx="6" cy="6.4" r="1.1" /><path d="M2.8 11.2l3-2.9 2.4 2.2 1.9-1.7 3.1 2.8" /></Line>
);
/** An open book: a skill the assistant reads. */
export const SkillIcon = (p: IconProps) => (
  <Line {...p} strokeLinecap="round" strokeLinejoin="round"><path d="M8 4.2C6.6 3.2 4.6 2.9 2.5 3.2v8.6c2.1-.3 4.1 0 5.5 1 1.4-1 3.4-1.3 5.5-1V3.2c-2.1-.3-4.1 0-5.5 1zM8 4.2v8.6" /></Line>
);
export const UploadIcon = (p: IconProps) => (
  <Line {...p} strokeLinecap="round" strokeLinejoin="round"><path d="M8 10.5V3M4.8 6.1L8 3l3.2 3.1M3 13h10" /></Line>
);
export const SendIcon = () => (
  <svg className="i-send" width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><path d="M8 13V3M3.5 7.5L8 3l4.5 4.5" /></svg>
);
export const StopIcon = () => (
  <svg className="i-stop" width="12" height="12" viewBox="0 0 12 12"><rect x="1" y="1" width="10" height="10" rx="2.5" fill="currentColor" /></svg>
);
export const ArrowDownIcon = () => (
  <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth={1.7} strokeLinecap="round"><path d="M6 2v8M2.5 6.5L6 10l3.5-3.5" /></svg>
);

/* artifact kinds, on an 18-unit grid */
function KindSvg({ children, ...rest }: SVGProps<SVGSVGElement>) {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinejoin="round" {...rest}>
      {children}
    </svg>
  );
}
const KIND_ICONS: Record<string, () => React.JSX.Element> = {
  html: () => <KindSvg><rect x="2.5" y="3.5" width="13" height="11" rx="2.5" /><path d="M2.5 7h13" /></KindSvg>,
  svg: () => <KindSvg><circle cx="6.5" cy="6.5" r="3.5" /><rect x="8" y="8" width="7" height="7" rx="1.5" /></KindSvg>,
  markdown: () => <KindSvg strokeLinecap="round"><path d="M4.5 2.5h6.5L14 5.5v10H4.5z" /><path d="M7 8.5h4.5M7 11.5h4.5" /></KindSvg>,
  code: () => <KindSvg strokeLinecap="round"><path d="M6.5 5.5L3 9l3.5 3.5M11.5 5.5L15 9l-3.5 3.5" /></KindSvg>,
  report: () => <KindSvg strokeLinecap="round"><path d="M4.5 2.5h6.5L14 5.5v10H4.5z" /><path d="M7 13V11M9.25 13V8.5M11.5 13v-3" /></KindSvg>,
};

export function KindIcon({ kind }: { kind?: string }) {
  const Icon = KIND_ICONS[kind ?? ''] ?? KIND_ICONS.code;
  return <Icon />;
}
