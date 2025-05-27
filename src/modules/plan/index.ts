import PlanModuleService from "./service";
import { Module } from "@medusajs/framework/utils";

export const PLAN_MODULE = "plan";

export default Module(PLAN_MODULE, {
  service: PlanModuleService,
});
