import { createProductVariantsWorkflow } from "@medusajs/medusa/core-flows"
import { STRIPE_PRODUCT_MODULE } from "../modules/stripe_product"
import StripeProductModuleService from "../modules/stripe_product/service"
import type { CreateProductParams } from '../modules/stripe_product/types'

createProductVariantsWorkflow.hooks.productVariantsCreated(
    async ({ product_variants, additional_data }, { container }) => {
        const logger = container.resolve("logger")
        
        try {
            logger.info('Executing productVariantCreated hook')
            
            // Resolve Stripe module service
            let stripeProductModuleService: StripeProductModuleService
            try {
                stripeProductModuleService = container.resolve(STRIPE_PRODUCT_MODULE)
                logger.info('stripe_product module resolved successfully')
            } catch (moduleError) {
                logger.error('Failed to resolve stripe_product module:', moduleError)
                throw new Error(`Module resolution failed: ${moduleError.message}`)
            }

            // Validate input data
            if (!product_variants || !Array.isArray(product_variants) || product_variants.length === 0) {
                logger.warn('No product variants provided or invalid data structure')
                return
            }

            let payloads: CreateProductParams[] = []
            logger.info('Building create product params...')

            // Build Stripe product parameters with error handling
            try {
                product_variants.forEach((variant, index) => {
                    try {
                        // Validate variant data
                        if (!variant) {
                            logger.warn(`Variant at index ${index} is null or undefined, skipping`)
                            return
                        }

                        if (!variant.title) {
                            logger.warn(`Variant at index ${index} missing title, using fallback`)
                        }

                        // Debug logging for variant structure
                        logger.info(`Processing variant ${index}: ${JSON.stringify({
                            id: variant.id,
                            title: variant.title,
                            product_id: variant.product_id,
                            prices_type: typeof variant.prices,
                            prices_is_array: Array.isArray(variant.prices),
                            prices_length: variant.prices?.length || 'N/A'
                        }, null, 2)}`)

                        const params: CreateProductParams = {
                            name: variant.title || `Product Variant ${index + 1}`,
                            description: variant.product?.description ?? "No description",
                            images: variant?.product?.thumbnail ? [variant.product.thumbnail] : [],
                            metadata: {
                                "medusa_variant_id": variant.id || "",
                                "medusa_product_id": variant?.product_id ?? ""
                            },
                            active: true,
                            // Ensure money_amounts is always an array or undefined
                            money_amounts: Array.isArray(variant.prices) ? variant.prices : undefined
                        }

                        logger.info(`Created payload for variant: ${params.name}`)
                        payloads.push(params)
                    } catch (variantError) {
                        logger.error(`Error processing variant at index ${index}:`, variantError)
                        // Continue processing other variants instead of failing completely
                    }
                })

                logger.info(`Total payloads created: ${payloads.length}`)
            } catch (payloadError) {
                logger.error('Error building product payloads:', payloadError)
                throw new Error(`Payload creation failed: ${payloadError.message}`)
            }

            // Early exit if no valid payloads were created
            if (payloads.length === 0) {
                logger.warn('No valid payloads created, skipping Stripe product creation')
                return
            }

            // Create Stripe products with proper async handling and error management
            const results = await Promise.allSettled(
                payloads.map(async (payload, index) => {
                    try {
                        logger.info(`Creating Stripe product for payload ${index + 1}/${payloads.length}: ${payload.name}`)
                        
                        let product
                        
                        // Use different methods based on whether money_amounts exist
                        if (payload.money_amounts && Array.isArray(payload.money_amounts) && payload.money_amounts.length > 0) {
                            logger.info(`Creating product with ${payload.money_amounts.length} money amounts`)
                            const result = await stripeProductModuleService.createProductWithPricesFromMoneyAmounts(payload)
                            product = result.product
                            logger.info(`Created product with ${result.prices.length} prices`)
                        } else {
                            logger.info('Creating basic product without specific pricing')
                            product = await stripeProductModuleService.createStripeProduct(payload)
                        }
                        
                        if (product?.id) {
                            logger.info(`Successfully created Stripe product: ${product.id} for variant: ${payload.name}`)
                            return {
                                success: true,
                                productId: product.id,
                                variantName: payload.name,
                                medusaVariantId: payload.metadata?.medusa_variant_id
                            }
                        } else {
                            throw new Error('Product creation returned invalid response')
                        }
                    } catch (productError) {
                        logger.error(`Failed to create Stripe product for variant "${payload.name}":`, productError)
                        return {
                            success: false,
                            error: productError.message,
                            variantName: payload.name,
                            medusaVariantId: payload.metadata?.medusa_variant_id
                        }
                    }
                })
            )

            // Process results and log summary
            const successful = results.filter(result => result.status === 'fulfilled' && result.value.success)
            const failed = results.filter(result => result.status === 'rejected' || (result.status === 'fulfilled' && !result.value.success))

            logger.info(`Stripe product creation summary: ${successful.length} successful, ${failed.length} failed out of ${results.length} total`)

            // Log successful creations
            successful.forEach(result => {
                if (result.status === 'fulfilled' && result.value.success) {
                    logger.info(`✅ Created: ${result.value.productId} for variant: ${result.value.variantName}`)
                }
            })

            // Log failures with details
            failed.forEach(result => {
                if (result.status === 'rejected') {
                    logger.error(`❌ Rejected: ${result.reason}`)
                } else if (result.status === 'fulfilled' && !result.value.success) {
                    logger.error(`❌ Failed: ${result.value.variantName} - ${result.value.error}`)
                }
            })

            // Optional: throw error if too many failures (adjust threshold as needed)
            const failureRate = failed.length / results.length
            if (failureRate > 0.5) { // More than 50% failed
                logger.error(`High failure rate detected: ${Math.round(failureRate * 100)}% of Stripe product creations failed`)
                // Uncomment the next line if you want to throw an error for high failure rates
                // throw new Error(`Too many Stripe product creation failures: ${failed.length}/${results.length}`)
            }

        } catch (globalError) {
            logger.error('Critical error in productVariantCreated hook:', globalError)

            // Decide whether to re-throw or handle gracefully
            // Re-throwing will cause the workflow to fail
            // Handling gracefully allows the workflow to continue
            
            // Option 1: Re-throw to fail the workflow
            throw globalError
            
            // Option 2: Handle gracefully (comment out the throw above and uncomment below)
            // logger.error('Hook failed but continuing workflow execution')
            // return { success: false, error: globalError.message }
        }
    }
)