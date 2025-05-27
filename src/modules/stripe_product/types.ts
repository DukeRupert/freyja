import { MoneyAmountDTO } from "@medusajs/framework/types"

export type ModuleOptions = {
    api_secret: string
    webhook_secret: string
}

export interface CreateProductParams {
    name: string
    description?: string
    images?: string[]
    metadata?: Record<string, string>
    active?: boolean
    default_price_data?: {
        currency: string
        unit_amount: number
    }
    money_amounts?: MoneyAmountDTO[]
}