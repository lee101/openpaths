import React, { useEffect } from 'react';

type AdSenseSlotProps = {
  slot?: string;
  className?: string;
};

export function AdSenseSlot({ slot = '7003733604', className = '' }: AdSenseSlotProps) {
  useEffect(() => {
    try {
      ((window as any).adsbygoogle = (window as any).adsbygoogle || []).push({});
    } catch {
      // Ad blockers and local dev can make AdSense unavailable.
    }
  }, [slot]);

  return (
    <div className={`mx-auto w-full max-w-6xl px-6 py-8 ${className}`}>
      <ins
        className="adsbygoogle"
        style={{ display: 'block' }}
        data-ad-client="ca-pub-8598649123553748"
        data-ad-slot={slot}
        data-ad-format="auto"
        data-full-width-responsive="true"
      />
    </div>
  );
}
