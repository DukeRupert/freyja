import { MedusaService } from "@medusajs/framework/utils"
import { InferTypeOf, DAL, Logger, MoneyAmountDTO } from "@medusajs/framework/types"
import Stripe from "stripe"
import StripeProduct from "./models/stripe_product"
import type { ModuleOptions, CreateProductParams } from "./types"

type StripeProductType = InferTypeOf<typeof StripeProduct>

type InjectedDependencies = {
  logger: Logger
  stripeProductRepository: DAL.RepositoryService<StripeProductType>
}

class StripeProductModuleService extends MedusaService({
  StripeProduct,
}) {
  protected options_: ModuleOptions
  private logger_: Logger
  protected stripeProductRepository_: DAL.RepositoryService<StripeProductType>
  private stripe_: Stripe

  constructor({ logger, stripeProductRepository }: InjectedDependencies, options?: ModuleOptions) {
    super(...arguments)
    this.logger_ = logger
    this.stripeProductRepository_ = stripeProductRepository
    this.options_ = options || {
      api_secret: "supersecret",
      webhook_secret: "verysecret"
    }

    // Initialize Stripe with your secret key
    this.stripe_ = new Stripe(this.options_.api_secret)
  }

  private async sendRequest(url: string, method: string, data?: any) {
    this.logger_.info(`Sending a ${method} request to ${url}.`)
    this.logger_.info(`Request Data: ${JSON.stringify(data, null, 2)}`)
    this.logger_.info(`API Key: ${JSON.stringify(this.options_.api_secret, null, 2)}`)
  }

  async createProduct(id: string) {
    await this.sendRequest("/brands", "POST", id)
  }

  /**
   * Maps MoneyAmountDTO array to Stripe price parameters
   * @param moneyAmounts - Array of MoneyAmountDTO
   * @param isRecurring - Whether the price should be recurring (default: true for monthly)
   * @returns Array of Stripe price parameters or default price
   */
  private mapMoneyAmountsToStripePrices(
    moneyAmounts?: MoneyAmountDTO[],
    isRecurring: boolean = true
  ): any[] {
    const defaultPriceParams = {
      currency: 'usd',
      unit_amount: 1000,
      recurring: {
        interval: 'month',
      }
    }

    // If no money amounts or empty array, return default
    if (!moneyAmounts || !Array.isArray(moneyAmounts) || moneyAmounts.length === 0) {
      this.logger_.info('Using default price parameters')
      return [defaultPriceParams]
    }

    try {
      // Map each money amount to Stripe price format
      return moneyAmounts.map((moneyAmount, index) => {
        try {
          if (!moneyAmount) {
            this.logger_.warn(`Money amount at index ${index} is null/undefined, skipping`)
            return null
          }

          const priceParams: any = {
            currency: (moneyAmount.currency_code || 'usd').toLowerCase(),
            // Convert to smallest currency unit (cents for USD)
            // Using numeric value which should already be in the correct format
            unit_amount: Math.round((() => {
              if (typeof moneyAmount.amount === 'string') {
                return parseFloat(moneyAmount.amount) || 10
              }
              if (typeof moneyAmount.amount === 'number') {
                return moneyAmount.amount
              }
              return 10
            })() * 100),
          }

          // Add recurring interval if specified
          if (isRecurring) {
            priceParams.recurring = {
              interval: 'month', // Default to monthly, could be made configurable
            }
          }

          return priceParams
        } catch (itemError) {
          this.logger_.error(`Error processing money amount at index ${index}:`, itemError)
          return null
        }
      }).filter(Boolean) // Remove null entries
    } catch (mapError) {
      this.logger_.error('Error mapping money amounts:', mapError)
      this.logger_.info('Falling back to default price parameters')
      return [defaultPriceParams]
    }
  }

  /**
   * Create a new product in Stripe
   * @param productData - The product data to create
   * @returns The created Stripe product
   */
  async createStripeProduct(productData: CreateProductParams): Promise<Stripe.Product> {
    try {
      this.logger_.info(`Creating Stripe product: ${productData.name}`)

      const stripeProduct = await this.stripe_.products.create({
        name: productData.name,
        description: productData.description,
        images: productData.images,
        metadata: productData.metadata,
        active: productData.active ?? true,
        default_price_data: productData.default_price_data,
      })

      this.logger_.info(`Successfully created Stripe product with ID: ${stripeProduct.id}`)

      // Optionally, save the product to your local database
      const localProduct = await this.stripeProductRepository_.create({
        stripe_id: stripeProduct.id,
        name: stripeProduct.name,
        description: stripeProduct.description,
        // Add other fields as needed based on your StripeProduct model
      })

      return stripeProduct
    } catch (error) {
      this.logger_.error(`Failed to create Stripe product: ${error.message}`)
      throw new Error(`Failed to create Stripe product: ${error.message}`)
    }
  }

  /**
   * Create a product with multiple prices based on money amounts
   * @param productData - The product data including money amounts
   * @param isRecurring - Whether prices should be recurring
   * @returns The created Stripe product and associated prices
   */
  async createProductWithPricesFromMoneyAmounts(
    productData: CreateProductParams,
    isRecurring: boolean = true
  ): Promise<{ product: Stripe.Product; prices: Stripe.Price[] }> {
    try {
      this.logger_.info(`Creating Stripe product with money amounts: ${productData.name}`)

      // Create product first
      const product = await this.stripe_.products.create({
        name: productData.name,
        description: productData.description,
        images: productData.images,
        metadata: productData.metadata,
        active: productData.active ?? true,
      })

      // Get price parameters from money amounts
      const priceParams = this.mapMoneyAmountsToStripePrices(productData.money_amounts, isRecurring)

      // Create prices for the product
      const prices: Stripe.Price[] = []
      for (const priceParam of priceParams) {
        const price = await this.stripe_.prices.create({
          product: product.id,
          ...priceParam,
        })
        prices.push(price)
      }

      this.logger_.info(`Successfully created Stripe product ${product.id} with ${prices.length} price(s)`)

      // Optionally, save to local database
      await this.stripeProductRepository_.create({
        stripe_id: product.id,
        name: product.name,
        description: product.description,
        // Add other fields as needed
      })

      return { product, prices }
    } catch (error) {
      this.logger_.error(`Failed to create Stripe product with money amounts: ${error.message}`)
      throw new Error(`Failed to create Stripe product with money amounts: ${error.message}`)
    }
  }

  /**
   * Retrieve a Stripe product by ID
   * @param productId - The Stripe product ID
   * @returns The Stripe product
   */
  async getStripeProduct(productId: string): Promise<Stripe.Product> {
    try {
      const product = await this.stripe_.products.retrieve(productId)
      return product
    } catch (error) {
      this.logger_.error(`Failed to retrieve Stripe product ${productId}: ${error.message}`)
      throw new Error(`Failed to retrieve Stripe product: ${error.message}`)
    }
  }

  /**
   * Update a Stripe product
   * @param productId - The Stripe product ID
   * @param updateData - The data to update
   * @returns The updated Stripe product
   */
  async updateStripeProduct(
    productId: string,
    updateData: Partial<CreateProductParams>
  ): Promise<Stripe.Product> {
    try {
      const updatedProduct = await this.stripe_.products.update(productId, {
        name: updateData.name,
        description: updateData.description,
        images: updateData.images,
        metadata: updateData.metadata,
        active: updateData.active,
      })

      this.logger_.info(`Successfully updated Stripe product: ${productId}`)
      return updatedProduct
    } catch (error) {
      this.logger_.error(`Failed to update Stripe product ${productId}: ${error.message}`)
      throw new Error(`Failed to update Stripe product: ${error.message}`)
    }
  }
}

export default StripeProductModuleService