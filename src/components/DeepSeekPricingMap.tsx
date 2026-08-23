import { useEffect, useMemo, useRef, useState, type MouseEvent } from 'react';
import { Locate, MapPin, Moon, Sun, Zap } from 'lucide-react';

// Compact equirectangular world outline (SVG path, viewBox 0 0 1000 500).
// Generated from Natural Earth 110m land, simplified for inline embedding.
const LAND_PATH = 'M335 472L333 475L316 473L335 472ZM58 471L45 468L52 468L58 471ZM375 467L378 468L380 472L360 475L350 474L365 467L375 467ZM163 454L170 454L166 456L159 455L163 454ZM225 450L233 451L220 451L216 450L225 450ZM310 447L309 450L303 451L292 449L300 448L301 443L305 441L310 447ZM337 428L328 430L326 432L327 434L318 439L328 446L331 455L304 463L285 463L295 466L284 468L283 470L338 481L362 477L381 478L421 473L418 470L401 471L402 467L451 459L456 457L454 455L457 453L471 448L479 449L481 447L499 449L522 444L530 447L537 444L575 446L589 443L594 440L607 444L651 433L671 439L691 439L694 442L688 445L692 446L689 450L694 451L705 444L716 443L730 437L741 437L744 434L749 437L766 437L777 437L786 432L795 436L816 433L833 437L874 434L875 431L882 436L904 436L913 440L929 440L949 446L976 449L970 455L961 457L954 462L958 467L964 469L949 470L944 475L971 483L1000 485L1000 500L0 500L0 485L3 484L28 483L61 487L102 486L73 482L75 478L64 475L82 476L93 473L69 470L61 467L60 464L80 465L94 462L94 459L97 459L183 455L188 458L220 459L222 458L215 456L212 452L232 454L250 454L252 452L274 455L277 453L292 455L313 451L310 444L312 437L325 430L339 426L341 426L337 428ZM312 400L319 402L308 404L293 397L302 400L307 396L312 400ZM337 392L339 394L330 394L337 392ZM695 388L691 388L691 385L696 386L695 388ZM904 363L912 364L911 370L906 371L902 364L904 363ZM981 364L984 365L981 372L976 373L970 380L963 378L964 375L981 364ZM985 350L991 355L996 355L987 366L986 361L983 360L985 354L980 346L985 350ZM995 298L996 300L993 300L995 298ZM998 297L996 296L1000 295L998 297ZM639 288L640 294L638 294L631 319L626 321L622 319L620 311L623 306L623 295L633 291L637 283L639 288ZM899 288L904 292L907 303L913 307L925 322L925 338L917 354L906 358L903 355L899 358L891 356L888 350L884 349L884 346L880 348L883 341L878 347L873 341L865 337L850 339L843 344L833 344L828 347L820 345L821 338L815 323L817 323L815 318L817 310L817 313L824 308L836 305L849 290L853 288L860 292L863 285L868 284L868 281L876 284L879 283L880 284L876 292L889 299L894 292L895 281L899 288ZM846 278L843 278L844 276L854 273L846 278ZM828 272L831 274L824 275L828 272ZM802 269L808 268L821 273L801 272L793 269L795 266L802 269ZM874 267L873 269L874 265L874 267ZM922 265L917 268L912 266L919 265L921 262L923 262L922 265ZM862 259L863 261L855 259L862 259ZM873 253L873 258L876 259L884 255L902 261L910 267L909 271L919 279L911 278L902 271L896 276L886 272L882 273L885 270L883 265L871 260L869 261L867 258L871 256L863 253L868 251L873 253ZM848 246L844 249L834 249L836 254L843 252L838 255L842 265L840 265L841 262L837 263L836 257L834 258L835 265L832 265L830 258L833 248L836 246L848 246ZM857 247L856 252L854 245L857 247ZM794 266L785 262L765 235L771 235L788 250L787 252L795 259L794 266ZM827 245L831 247L827 248L823 261L806 258L803 251L805 244L809 245L809 243L814 241L824 231L831 235L826 241L827 245ZM851 227L848 234L843 228L839 230L843 226L849 225L848 223L851 227ZM726 233L722 231L723 223L727 229L726 233ZM331 222L328 222L330 220L331 222ZM844 221L842 225L840 223L842 220L843 221L845 219L844 221ZM829 224L825 227L832 218L829 224ZM839 217L842 218L839 221L839 217ZM849 216L849 219L847 219L847 222L845 215L849 216ZM838 214L837 216L834 213L838 214ZM837 199L840 199L840 203L838 210L844 212L845 215L835 212L836 210L834 208L833 205L837 199ZM298 195L310 198L302 201L295 200L293 198L299 198L296 195L298 195ZM806 198L802 199L802 196L808 194L806 198ZM68 197L67 194L70 196L68 197ZM279 187L294 194L284 195L286 193L281 190L273 187L264 189L271 186L279 187ZM837 187L835 189L834 185L837 180L837 187ZM286 176L286 178L284 175L286 176ZM874 155L873 158L868 158L869 155L874 155ZM596 151L592 154L590 152L596 151ZM566 151L573 152L569 153L566 151ZM543 144L542 148L535 146L543 144ZM526 136L527 141L524 142L523 136L526 136ZM892 147L890 152L877 157L875 154L864 156L867 158L865 163L862 163L862 160L859 158L868 152L877 151L880 146L882 148L887 144L890 136L893 135L894 139L892 147ZM527 133L526 135L524 133L526 131L527 133ZM900 127L904 127L904 130L898 133L893 131L892 134L889 135L894 123L900 127ZM157 115L151 114L143 109L151 110L157 115ZM344 109L342 112L351 113L353 120L350 120L349 117L346 120L344 118L335 118L341 109L345 107L344 109ZM131 100L134 100L133 103L136 105L130 102L131 100ZM899 109L902 114L898 113L896 117L899 122L897 120L895 122L894 102L896 99L899 109ZM481 105L472 106L475 103L473 100L481 97L484 98L481 105ZM535 96L534 98L530 95L535 96ZM75 91L72 92L70 90L76 89L75 91ZM492 87L489 90L495 90L491 95L505 104L504 108L485 111L491 107L485 106L488 105L487 101L492 100L487 98L486 95L484 96L483 92L486 87L492 87ZM263 68L277 73L258 73L263 68ZM460 65L462 69L448 74L437 72L440 71L433 70L438 68L432 68L439 66L443 67L460 65ZM289 63L285 62L289 60L291 62L289 63ZM14 65L23 64L28 67L21 68L20 72L5 68L4 66L0 67L0 70L0 58L14 63L14 65ZM234 58L223 57L227 55L234 58ZM1000 53L996 53L1000 51L1000 53ZM4 53L0 53L0 51L7 52L4 53ZM248 57L248 60L252 58L257 63L262 56L270 57L274 58L272 61L274 62L268 66L262 65L257 70L241 78L237 86L241 87L244 91L271 97L272 102L278 108L282 104L278 98L287 93L282 87L285 84L283 77L295 77L307 80L308 86L312 88L321 82L329 92L328 94L341 98L345 105L333 110L316 110L302 120L319 113L322 115L319 116L321 122L329 123L332 119L334 122L318 129L316 126L321 124L314 125L304 130L306 134L295 136L300 136L295 137L292 142L290 140L292 143L289 147L288 141L288 144L286 144L290 151L274 163L277 180L273 178L266 166L252 166L252 169L250 169L239 167L232 171L228 188L233 196L238 200L244 198L248 196L249 192L258 190L253 206L268 208L267 219L274 226L279 223L287 226L292 219L301 215L302 216L300 218L301 225L302 220L306 216L311 221L328 220L327 222L341 233L350 234L357 238L361 245L360 250L365 251L365 253L367 252L375 254L376 257L389 258L401 264L404 270L402 275L393 286L391 300L386 311L368 319L364 330L351 346L344 347L338 344L342 353L335 358L327 358L326 364L319 364L320 367L324 368L319 371L318 375L313 377L312 379L318 381L317 384L308 391L311 395L303 397L303 400L292 395L290 385L294 380L290 380L293 373L297 373L298 368L294 370L297 359L296 353L302 340L305 305L301 298L289 291L278 270L274 267L274 263L278 257L275 256L275 253L286 239L283 227L279 225L275 230L262 222L257 213L247 211L237 205L232 207L212 199L207 195L205 187L184 162L181 162L181 166L196 185L194 187L188 181L188 178L180 173L183 171L174 158L165 154L154 138L156 124L154 116L158 117L159 119L159 114L146 109L145 105L128 89L91 81L79 86L82 80L60 94L42 99L62 90L64 86L50 87L50 84L45 84L39 79L43 75L53 73L51 71L53 70L42 71L33 68L43 65L51 66L37 60L65 52L121 59L144 54L151 57L154 55L155 57L163 56L180 59L184 60L180 61L198 63L201 61L198 60L200 59L205 59L218 62L227 62L229 60L233 60L233 63L238 58L232 55L236 50L242 52L246 55L243 56L248 57ZM183 47L195 47L199 51L199 47L204 47L210 53L219 57L215 57L215 59L177 58L174 56L188 55L172 54L177 52L168 51L173 48L183 47ZM210 46L207 48L203 46L210 46ZM260 47L262 49L271 45L276 50L284 48L294 52L299 51L311 55L314 58L309 59L328 64L322 69L311 66L320 74L319 76L309 73L316 78L292 70L284 72L282 71L284 69L295 68L297 62L281 55L254 54L251 53L254 52L250 52L249 49L254 46L262 45L260 47ZM221 45L230 45L228 47L232 48L231 51L224 52L215 49L221 48L218 46L221 45ZM899 47L889 46L895 45L899 47ZM241 48L235 50L233 46L237 44L249 45L241 48ZM165 52L158 53L150 50L156 45L153 44L173 44L179 46L165 52ZM240 42L231 42L237 40L240 42ZM903 40L901 42L886 43L880 41L886 39L903 40ZM226 37L229 38L227 42L215 40L215 38L226 37ZM199 38L206 39L205 42L184 43L189 41L173 41L179 38L197 40L193 38L199 38ZM660 54L643 51L655 41L691 37L662 44L654 49L660 54ZM237 36L252 40L275 40L278 42L243 42L239 38L230 37L237 36ZM177 34L175 37L159 39L177 34ZM797 36L817 39L804 44L842 47L842 45L853 46L865 53L867 50L889 51L887 49L890 48L915 49L925 53L942 53L947 57L966 57L971 59L974 58L973 55L996 57L1000 58L1000 70L993 71L999 76L982 79L973 84L969 82L954 84L950 88L953 90L950 98L945 99L945 102L940 103L936 108L932 96L933 92L955 80L957 76L945 82L943 78L935 79L928 84L931 86L895 86L875 98L884 101L889 99L893 105L889 115L884 121L875 129L867 130L854 140L860 148L859 153L851 154L850 148L852 148L846 144L848 140L836 142L839 138L838 136L828 141L826 142L830 146L840 146L831 153L839 162L837 165L839 167L838 172L830 182L822 187L808 191L807 193L801 190L794 195L804 213L803 218L792 226L792 222L778 213L776 224L786 235L790 246L782 242L778 232L773 228L774 218L770 203L765 206L762 205L762 199L754 187L751 187L751 189L742 190L740 194L723 206L722 221L715 228L704 206L702 191L696 192L684 179L659 179L657 175L652 176L643 173L639 166L633 167L641 181L642 178L643 178L644 183L650 183L657 177L658 183L666 188L661 194L660 197L654 202L635 211L621 215L618 203L609 191L607 184L596 172L597 168L594 173L590 167L599 186L602 189L604 198L620 216L619 217L624 221L642 217L642 220L633 238L612 257L609 263L608 268L612 280L613 291L610 296L597 305L599 316L590 321L589 330L578 341L572 344L554 347L551 345L551 338L542 325L540 311L533 300L533 294L538 280L533 264L524 253L526 240L524 237L516 238L512 233L495 237L475 237L465 230L454 216L451 209L455 200L453 189L460 177L473 167L474 160L484 151L494 152L504 148L526 146L531 147L529 156L553 166L556 160L560 159L580 164L586 162L594 164L600 154L600 148L577 148L573 140L581 135L593 133L607 136L616 133L602 124L609 119L597 121L601 125L594 127L590 124L592 122L585 121L577 132L580 136L573 138L569 136L566 137L566 139L563 138L567 145L564 145L564 149L562 149L554 138L554 134L537 123L534 124L535 128L551 138L547 138L547 142L545 144L543 139L525 127L518 130L509 130L508 134L502 136L500 142L494 148L485 150L482 147L475 148L474 130L496 128L497 122L487 115L496 115L495 112L504 111L513 103L523 101L524 100L524 91L529 90L530 93L527 96L530 100L555 99L559 97L560 91L567 92L568 88L565 86L581 83L564 84L559 81L560 74L571 69L566 67L562 67L559 71L550 76L548 80L552 83L547 87L544 94L536 96L529 85L523 88L516 87L514 78L529 71L541 62L568 53L578 52L587 54L583 55L586 57L612 61L614 64L611 66L592 65L597 67L597 71L603 73L601 70L603 69L610 71L612 70L610 68L617 65L622 66L624 65L621 60L628 60L630 62L627 64L629 65L649 59L651 59L649 61L663 59L667 60L670 58L667 57L668 56L690 61L692 59L686 57L685 53L692 48L702 48L700 52L702 54L702 58L705 60L698 66L701 66L708 62L708 58L704 57L707 54L703 52L708 50L707 48L710 49L709 52L712 52L711 50L715 49L726 51L724 45L741 45L739 43L742 41L780 38L790 34L797 36ZM636 135L640 138L636 142L637 146L650 147L650 142L646 139L647 136L652 136L649 133L647 136L646 131L640 126L647 124L647 120L636 121L630 126L636 135ZM569 34L558 34L564 32L569 34ZM234 33L227 33L226 31L234 33ZM222 32L208 32L211 31L207 30L222 32ZM792 32L776 34L784 30L792 32ZM551 29L560 31L544 37L529 29L551 29ZM571 27L576 28L564 29L548 27L571 27ZM778 31L764 30L753 27L767 24L778 28L778 31ZM258 29L262 30L248 33L231 27L243 24L258 29ZM310 19L328 21L312 24L318 24L302 28L286 30L291 32L278 36L284 37L276 38L251 38L256 36L255 34L264 35L256 32L264 30L259 27L273 26L257 26L246 23L280 19L310 19ZM425 18L442 20L413 22L466 24L444 27L451 27L445 31L445 34L449 36L440 37L445 39L446 41L443 41L446 44L435 46L438 49L431 49L440 54L429 52L430 53L427 55L438 55L389 68L381 76L380 83L366 81L357 73L350 63L359 56L348 57L349 53L357 54L345 51L348 48L337 40L310 39L302 36L315 35L296 33L317 29L311 27L326 23L360 21L376 23L370 20L379 19L425 18Z';

const W = 1000;
const H = 500;

// DeepSeek peak windows in minutes-of-UTC-day.
const PEAK_WINDOWS: Array<[number, number]> = [
  [1 * 60, 4 * 60],
  [6 * 60, 10 * 60],
];

// Weekend off-peak rule (DeepSeek): effective 00:00 Beijing (UTC+8) on
// Sunday 2026-08-23, i.e. 2026-08-22T16:00:00Z.
const WEEKEND_OFF_PEAK_AT = Date.parse('2026-08-22T16:00:00Z');

// Approximate city markers: [name, lat, lon, ianaTz].
const CITIES: Array<[string, number, number, string]> = [
  ['Auckland', -36.85, 174.76, 'Pacific/Auckland'],
  ['Sydney', -33.87, 151.21, 'Australia/Sydney'],
  ['Tokyo', 35.68, 139.69, 'Asia/Tokyo'],
  ['Beijing', 39.9, 116.4, 'Asia/Shanghai'],
  ['Dubai', 25.2, 55.27, 'Asia/Dubai'],
  ['Berlin', 52.52, 13.4, 'Europe/Berlin'],
  ['London', 51.5, -0.13, 'Europe/London'],
  ['New York', 40.71, -74.0, 'America/New_York'],
  ['San Francisco', 37.77, -122.42, 'America/Los_Angeles'],
];

function clampLat(lat: number): number {
  return Math.max(-85, Math.min(85, lat));
}

function xOfLon(lon: number): number {
  return ((lon + 180) / 360) * W;
}

function yOfLat(lat: number): number {
  return ((90 - clampLat(lat)) / 180) * H;
}

function inWindow(dayMinute: number, windows: Array<[number, number]>): boolean {
  for (const [s, e] of windows) {
    if (dayMinute >= s && dayMinute < e) return true;
  }
  return false;
}

function beijingWeekend(now: Date): boolean {
  const bj = now.getTime() + 8 * 3600 * 1000;
  const day = new Date(bj).getUTCDay(); // 0 Sun .. 6 Sat
  return day === 0 || day === 6;
}

function beijingWeekendActive(now: Date): boolean {
  return now.getTime() >= WEEKEND_OFF_PEAK_AT && beijingWeekend(now);
}

// Global pricing state: the actual rate billed for any request right now.
function globalIsPeak(now: Date): boolean {
  if (beijingWeekendActive(now)) return false;
  const m = now.getUTCHours() * 60 + now.getUTCMinutes();
  return inWindow(m, PEAK_WINDOWS);
}

// Approximate local mean time (UTC + lon/15) for a given longitude.
function regionMinute(now: Date, lon: number): number {
  const utcMin = now.getUTCHours() * 60 + now.getUTCMinutes();
  const off = Math.round(lon / 15) * 60;
  return (((utcMin + off) % 1440) + 1440) % 1440;
}

function regionIsPeak(now: Date, lon: number): boolean {
  if (beijingWeekendActive(now)) return false;
  return inWindow(regionMinute(now, lon), PEAK_WINDOWS);
}

function declinationDeg(now: Date): number {
  const start = Date.UTC(now.getUTCFullYear(), 0, 0);
  const doy = Math.floor((now.getTime() - start) / 86400000);
  return 23.44 * Math.sin((2 * Math.PI / 365) * (doy - 81));
}

// Subsolar longitude at this instant (degrees, in [-180, 180)).
function subsolarLon(now: Date): number {
  const h = now.getUTCHours() + now.getUTCMinutes() / 60 + now.getUTCSeconds() / 3600;
  let lon = 180 - h * 15;
  while (lon <= -180) lon += 360;
  while (lon > 180) lon -= 360;
  return lon;
}

function cityTime(tz: string, d: Date): string {
  try {
    return new Intl.DateTimeFormat('en-GB', {
      timeZone: tz,
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    }).format(d);
  } catch {
    return '--';
  }
}

function pricingAt(lon: number, lat: number, now: Date): { peak: boolean; half: boolean } {
  const peak = regionIsPeak(now, lon);
  return { peak, half: !peak };
}

export function DeepSeekPricingMap() {
  const [now, setNow] = useState(() => new Date());
  const svgRef = useRef<SVGSVGElement>(null);
  const [user, setUser] = useState<{ lat: number; lon: number; name: string } | null>(null);

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);

  useEffect(() => {
    if (!('geolocation' in navigator)) {
      setUser({ lat: -36.85, lon: 174.76, name: 'Auckland' });
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => setUser({ lat: pos.coords.latitude, lon: pos.coords.longitude, name: 'You' }),
      () => setUser({ lat: -36.85, lon: 174.76, name: 'Auckland' }),
      { timeout: 6000 },
    );
  }, []);

  // Recompute the geo/pricing model once per minute.
  const minuteKey = Math.floor(now.getTime() / 60000);
  const model = useMemo(() => {
    const peakGlobal = globalIsPeak(now);
    const decl = (declinationDeg(now) * Math.PI) / 180;
    const slon = subsolarLon(now);
    const northNight = declinationDeg(now) < 0;
    const g: {
      peakGlobal: boolean;
      decl: number;
      slon: number;
      northNight: boolean;
      boundary: Array<[number, number]>;
      nightD: string;
      bands: Array<{ x: number; w: number; peak: boolean }>;
    } = { peakGlobal, decl, slon, northNight, boundary: [], nightD: '', bands: [] };

    // Day/night terminator boundary (single-valued in longitude).
    const boundary: Array<[number, number]> = [];
    for (let lon = -180; lon <= 180; lon += 2) {
      const H = ((lon - slon) * Math.PI) / 180;
      const cosD = Math.cos(decl);
      const sinD = Math.sin(decl);
      const phiT =
        Math.abs(sinD) < 1e-5 ? 0 : Math.atan(-(cosD * Math.cos(H)) / sinD);
      const phiDeg = (phiT * 180) / Math.PI;
      boundary.push([xOfLon(lon), yOfLat(phiDeg)]);
    }
    g.boundary = boundary;

    // Night polygon: side of the terminator that faces away from the sun.
    const edgeY = northNight ? 0 : H;
    let d = 'M' + boundary.map((p) => `${p[0].toFixed(1)} ${p[1].toFixed(1)}`).join('L');
    d += ` L${W} ${edgeY} L0 ${edgeY} Z`;
    g.nightD = d;

    // Cost bands: green = off-peak (1/2 price), red = peak (full price).
    const bands: Array<{ x: number; w: number; peak: boolean }> = [];
    for (let lon = -180; lon < 180; lon += 2) {
      bands.push({ x: (lon + 180) / 360 * W, w: (2 / 360) * W, peak: regionIsPeak(now, lon) });
    }
    g.bands = bands;

    return g;
  }, [minuteKey, now]);

  const utcH = now.getUTCHours();
  const utcM = now.getUTCMinutes();
  const utcS = now.getUTCSeconds();

  const timeline = useMemo(() => {
    const items: Array<{ label: string; peak: boolean; isNow: boolean }> = [];
    for (let i = 0; i < 24; i++) {
      const t = new Date(Math.floor(now.getTime() / 3600000) * 3600000 + i * 3600000);
      items.push({
        label: new Intl.DateTimeFormat('en-GB', { hour: '2-digit', hour12: false }).format(t),
        peak: globalIsPeak(t),
        isNow: i === 0,
      });
    }
    return items;
  }, [minuteKey, now]);

  function onMapClick(e: MouseEvent<SVGSVGElement>) {
    const svg = svgRef.current;
    if (!svg) return;
    const rect = svg.getBoundingClientRect();
    const x = ((e.clientX - rect.left) / rect.width) * W;
    const y = ((e.clientY - rect.top) / rect.height) * H;
    const lon = (x / W) * 360 - 180;
    const lat = 90 - (y / H) * 180;
    setUser({ lat, lon, name: 'You (clicked)' });
  }

  const userLon = user?.lon ?? 174.76;
  const userLat = user?.lat ?? -36.85;
  const userOffset = Math.round(userLon / 15);
  const userMinute = regionMinute(now, userLon);
  const userPeak = regionIsPeak(now, userLon);
  const userLocal = `${String(Math.floor(userMinute / 60) % 24).padStart(2, '0')}:${String(userMinute % 60).padStart(2, '0')}`;

  const sunLon = model.slon;
  const sunLat = declinationDeg(now);

  return (
    <div className="my-8 rounded-2xl border border-white/15 bg-[#070d1a] p-4 sm:p-6">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="text-lg font-semibold tracking-tight text-white">
            DeepSeek peak / off-peak world clock
          </div>
          <div className="text-xs text-white/45">
            Green = half price · Red = peak · Dark = night · Fixed UTC window
          </div>
        </div>
        <div className="flex items-center gap-3">
          <span
            className={
              'inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium ' +
              (model.peakGlobal
                ? 'bg-red-500/15 text-red-300 ring-1 ring-red-400/30'
                : 'bg-emerald-500/15 text-emerald-300 ring-1 ring-emerald-400/30')
            }
          >
            {model.peakGlobal ? <Zap className="h-3.5 w-3.5" /> : <Sun className="h-3.5 w-3.5" />}
            {model.peakGlobal ? 'Peak – full price' : 'Off-peak – half price'}
          </span>
          <span className="font-mono text-sm text-white/70 tabular-nums">
            {String(utcH).padStart(2, '0')}:{String(utcM).padStart(2, '0')}:{String(utcS).padStart(2, '0')} UTC
          </span>
        </div>
      </div>

      <div className="relative overflow-hidden rounded-xl border border-white/10">
        <svg
          ref={svgRef}
          viewBox={`0 0 ${W} ${H}`}
          className="block w-full select-none"
          onClick={onMapClick}
          role="img"
          aria-label="World map of DeepSeek peak and off-peak pricing by timezone"
        >
          {/* price bands (green/red by local clock) — solid region base */}
          <g shapeRendering="crispEdges">
            {model.bands.map((b, i) => (
              <rect
                key={i}
                x={b.x}
                y={0}
                width={b.w + 0.3}
                height={H}
                fill={b.peak ? '#8a1d1d' : '#137a4b'}
              />
            ))}
          </g>

          {/* land — translucent silhouette so the price region reads through */}
          <path d={LAND_PATH} fill="#0a1728" fillOpacity={0.55} stroke="#aebfd8" strokeWidth={0.6} />

          {/* night side */}
          <path d={model.nightD} fill="#020508" fillOpacity={0.6} />

          {/* terminator line */}
          <path
            d={'M' + model.boundary.map((p) => `${p[0].toFixed(1)} ${p[1].toFixed(1)}`).join('L')}
            fill="none"
            stroke="#e2e8f0"
            strokeWidth={1.2}
            opacity={0.8}
            strokeDasharray="3 3"
          />

          {/* subsolar sun marker */}
          <circle cx={xOfLon(sunLon)} cy={yOfLat(sunLat)} r={8} fill="#fbbf24" opacity={0.9} />
          <circle cx={xOfLon(sunLon)} cy={yOfLat(sunLat)} r={16} fill="#fbbf24" opacity={0.25} />

          {/* city markers */}
          {CITIES.map(([name, lat, lon]) => {
            const peak = regionIsPeak(now, lon);
            return (
              <g key={name}>
                <circle cx={xOfLon(lon)} cy={yOfLat(lat)} r={3} fill="#e2e8f0" stroke="#0a1220" strokeWidth={0.6} />
                <text
                  x={xOfLon(lon) + 5}
                  y={yOfLat(lat) + 3}
                  fontSize={9}
                  fill={peak ? '#fca5a5' : '#86efac'}
                  className="pointer-events-none"
                >
                  {name}
                </text>
              </g>
            );
          })}

          {/* user marker */}
          {user && (
            <g>
              <circle cx={xOfLon(userLon)} cy={yOfLat(userLat)} r={7} fill="#38bdf8" stroke="#0a1220" strokeWidth={1.5} />
              <circle cx={xOfLon(userLon)} cy={yOfLat(userLat)} r={13} fill="#38bdf8" opacity={0.25} />
              <text x={xOfLon(userLon) + 9} y={yOfLat(userLat) - 4} fontSize={12} fontWeight={700} fill="#38bdf8">
                You · {userLocal} local
              </text>
            </g>
          )}
        </svg>
      </div>

      <div className="mt-3 grid gap-3 sm:grid-cols-[1fr_auto]">
        <div>
          <div className="mb-1 text-xs uppercase tracking-wider text-white/40">DeepSeek half-price timeline</div>
          <div className="flex h-8 overflow-hidden rounded-lg border border-white/10">
            {timeline.map((seg, i) => (
              <div
                key={i}
                title={`${seg.label}:00 — ${seg.peak ? 'peak (full price)' : 'off-peak (half price)'}`}
                className={'flex-1 ' + (seg.peak ? 'bg-red-500/40' : 'bg-emerald-500/35')}
              />
            ))}
          </div>
          <div className="mt-1 flex justify-between text-[10px] text-white/35">
            <span>{timeline[0]?.label}:00 now</span>
            <span>+24h</span>
          </div>
        </div>

        <div className="flex flex-col gap-2 sm:w-56">
          <div className="rounded-lg border border-white/10 bg-white/[0.04] p-3">
            <div className="text-xs text-white/45">Your window</div>
            <div className="mt-1 flex items-center gap-2 text-sm font-medium text-white">
              <MapPin className="h-4 w-4 text-sky-300" />
                            <span>
                {userLocal} local
              </span>
              <span className="text-xs text-white/50">
                ({userOffset >= 0 ? 'UTC+' + userOffset : 'UTC' + userOffset})
              </span>
            </div>
            <div className={'mt-1 text-xs ' + (userPeak ? 'text-red-300' : 'text-emerald-300')}>
              {userPeak ? 'Peak – full price' : 'Off-peak – half price'}
            </div>
          </div>
          <button
            onClick={() => {
              if ('geolocation' in navigator) {
                navigator.geolocation.getCurrentPosition(
                  (pos) => setUser({ lat: pos.coords.latitude, lon: pos.coords.longitude, name: 'You' }),
                  () => setUser({ lat: -36.85, lon: 174.76, name: 'Auckland' }),
                );
              }
            }}
            className="inline-flex items-center justify-center gap-1.5 rounded-lg border border-white/15 bg-white/[0.06] px-3 py-1.5 text-xs text-white/70 hover:bg-white/10"
          >
            <Locate className="h-3.5 w-3.5" /> Locate me
          </button>
        </div>
      </div>

      <div className="mt-3 flex items-start gap-2 text-xs text-white/40">
        <Moon className="mt-0.5 h-4 w-4 shrink-0 text-indigo-300" />
        <p>
          The shaded half marks where it is currently night. DeepSeek bills one shared peak window in UTC
          (01:00–04:00 and 06:00–10:00), so at any instant every request is priced the same worldwide; the green/red
          band shows what that fixed window reads on each region's local clock. Off-peak is exactly half of peak, and
          since the 23 Aug 2026 change the whole Beijing-time weekend is off-peak.
        </p>
      </div>
    </div>
  );
}
