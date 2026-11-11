import SiteHeader from "../components/SiteHeader";
import HeroSection from "../components/HeroSection";
import ProblemSection from "../components/ProblemSection";
import BenefitsSection from "../components/BenefitsSection";
import ServiceSection from "../components/ServiceSection";
import StepsSection from "../components/StepsSection";
import PublicBuildsSection from "../components/PublicBuildsSection";
import FAQSection from "../components/FAQSection";
import CallToAction from "../components/CallToAction";
import SiteFooter from "../components/SiteFooter";

export default function HomePage() {
  return (
    <>
      <SiteHeader />
      <main>
        <HeroSection />
        <BenefitsSection />
        <ServiceSection />
        <PublicBuildsSection />
        <ProblemSection />
        <StepsSection />
        <FAQSection />
        <CallToAction />
      </main>
      <SiteFooter />
    </>
  );
}
