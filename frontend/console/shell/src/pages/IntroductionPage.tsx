// frontend/shell/src/pages/IntroductionPage.tsx
import { useNavigate } from "react-router-dom";
import { Button } from "../shared/ui/button";
import "./IntroductionPage.css";

export default function IntroductionPage() {
  const navigate = useNavigate();

  return (
    <div className="intro-container">
      <h1 className="intro-title">Welcome to Narratives</h1>

      <div className="intro-buttons">
        <Button
          variant="outline"
          size="lg"
          onClick={() => alert("SNS へ移動します")}
        >
          SNS
        </Button>

        <Button
          variant="outline"
          size="lg"
          onClick={() => alert("Inspection へ移動します")}
        >
          Inspection
        </Button>

        <Button
          variant="solid"
          size="lg"
          onClick={() => navigate("/console")}
        >
          Console
        </Button>
      </div>
    </div>
  );
}
