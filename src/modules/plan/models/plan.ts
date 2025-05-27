import { model } from "@medusajs/framework/utils";

const Plan = model.define("plan", {
  id: model.id().primaryKey(),
  name: model.text(),
  active: model.boolean(),
  interval: model.text(),
  interval_count: model.number(),
});

export default Plan;
