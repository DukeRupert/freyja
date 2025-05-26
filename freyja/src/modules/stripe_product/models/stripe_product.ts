import { model } from "@medusajs/framework/utils"

const StripeProduct = model.define("stripe_product", {
    id: model.id().primaryKey(),
    name: model.text(),
    active: model.boolean(),
})

export default StripeProduct