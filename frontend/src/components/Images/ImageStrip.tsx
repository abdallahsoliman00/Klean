import { useStore } from '../../store';
import { CloseIcon, SpinnerIcon } from '../common/icons';

export interface StripImage {
  src: string;
  /** Present when the image can be taken off (the composer's draft). */
  remove?: () => void;
}

/** A row of thumbnails. Each opens large; in the composer each can also be removed. */
export function ImageStrip({ images, busy = 0, className = '' }: { images: StripImage[]; busy?: number; className?: string }) {
  const showImage = useStore((s) => s.showImage);
  const label = (i: number) => (images.length > 1 ? `image ${i + 1} of ${images.length}` : 'image');
  return (
    <ul className={('thumbs ' + className).trim()} data-count={Math.min(images.length, 3)}>
      {images.map((image, i) => (
        // keyed by position: the picture can swap from the local copy to the server's without a new element
        <li key={i} className="thumb">
          <button type="button" className="thumb-open" aria-label={`Open ${label(i)}`} onClick={() => showImage(image.src, `Attached ${label(i)}`)}>
            <img src={image.src} alt="" draggable={false} />
          </button>
          {image.remove && (
            <button type="button" className="thumb-x" aria-label={`Remove ${label(i)}`} title="Remove" onClick={image.remove}>
              <CloseIcon />
            </button>
          )}
        </li>
      ))}
      {Array.from({ length: busy }, (_, i) => (
        <li key={`busy-${i}`} className="thumb busy" role="status" aria-label="Adding image">
          <SpinnerIcon />
        </li>
      ))}
    </ul>
  );
}
