import {
  LoaderOptions,
} from "@medusajs/framework/types"
import type { ModuleOptions } from "../types"

export default async function stripeProductLoader({
  options,
}: LoaderOptions<ModuleOptions>) {

  console.log(
    "[stripe_product MODULE] Just started the Medusa application!",
    options
  )
}