import { LoaderOptions } from "@medusajs/framework/types"
import { MedusaError } from "@medusajs/framework/utils"
import type { ModuleOptions } from "../types"

export default async function validationLoader({
    options,
}: LoaderOptions<ModuleOptions>) {
    if (!options?.api_secret) {
        throw new MedusaError(
            MedusaError.Types.INVALID_DATA,
            "stripe_product Module requires an api_secret option."
        )
    }
}