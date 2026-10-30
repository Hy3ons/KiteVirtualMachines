import { LandingHeroCopy } from '../LandingHeroCopy';
import { DemoPublicHeader } from './DemoPublicHeader';

// DemoLanding returns to the shared homepage hero without recursively mounting the product demo.
export function DemoLanding() {
  return <div className="public-page demo-landing">
    <DemoPublicHeader authenticated />
    <div className="marketing-main"><section className="marketing-hero" aria-labelledby="demo-marketing-title"><div className="marketing-content"><LandingHeroCopy preview /></div></section></div>
  </div>;
}
