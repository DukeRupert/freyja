import StripeProductModuleService from "./service"
import { Module } from "@medusajs/framework/utils"
import validationLoader from "./loaders/validate"

export const STRIPE_PRODUCT_MODULE = "stripe_product"

export default Module(STRIPE_PRODUCT_MODULE, {
    service: StripeProductModuleService,
    loaders: [validationLoader],
})