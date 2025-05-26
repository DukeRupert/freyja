import { MedusaService } from "@medusajs/framework/utils"
import StripeProduct from "./models/stripe_product"

class StripeProductModuleService extends MedusaService({
  StripeProduct,
}){
}

export default StripeProductModuleService