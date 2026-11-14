import { ART_OVERLAY_QUERY, ART_WIDTH, MOBILE_QUERY } from '../config';

/** Too narrow for three columns: the artifact panel slides over the thread. */
export const artifactPanelOverlays = () => matchMedia(ART_OVERLAY_QUERY).matches;
/** The sidebar is a drawer. */
export const isMobile = () => matchMedia(MOBILE_QUERY).matches;
/** The widest the artifact panel may be dragged in this window. */
export const maxArtifactWidth = () => Math.round(Math.min(ART_WIDTH.max, innerWidth * ART_WIDTH.maxShare));
/** The artifact panel's width before the user has resized it. */
export const initialArtifactWidth = () => Math.round(Math.max(ART_WIDTH.min, Math.min(ART_WIDTH.initial.max, innerWidth * ART_WIDTH.initial.share)));
