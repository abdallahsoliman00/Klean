import { ART_OVERLAY_QUERY, MOBILE_QUERY } from '../config';

/** Too narrow for three columns: the artifact panel slides over the thread. */
export const artifactPanelOverlays = () => matchMedia(ART_OVERLAY_QUERY).matches;
/** The sidebar is a drawer. */
export const isMobile = () => matchMedia(MOBILE_QUERY).matches;
