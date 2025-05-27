import { MedusaService } from "@medusajs/framework/utils";
import Plan from "./models/plan";

class PlanModuleService extends MedusaService({
  Plan,
}) {}

export default PlanModuleService;
