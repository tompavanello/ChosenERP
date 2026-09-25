import { Hero } from "@/components/sections/hero";
import { Trust } from "@/components/sections/trust";
import { Problems } from "@/components/sections/problems";
import { Features } from "@/components/sections/features";
import { Governance } from "@/components/sections/governance";
import { MultiBranch } from "@/components/sections/multi-branch";
import { Security } from "@/components/sections/security";
import { Steps } from "@/components/sections/steps";
import { Testimonials } from "@/components/sections/testimonials";
import { Pricing } from "@/components/sections/pricing";
import { Faq } from "@/components/sections/faq";
import { FinalCta } from "@/components/sections/final-cta";

export default function HomePage() {
  return (
    <>
      <Hero />
      <Trust />
      <Problems />
      <Features />
      <Governance />
      <MultiBranch />
      <Security />
      <Steps />
      <Testimonials />
      <Pricing />
      <Faq />
      <FinalCta />
    </>
  );
}
