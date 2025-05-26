import { Migration } from '@mikro-orm/migrations';

export class Migration20250526224201 extends Migration {

  override async up(): Promise<void> {
    this.addSql(`create table if not exists "stripe_product" ("id" text not null, "name" text not null, "active" boolean not null, "created_at" timestamptz not null default now(), "updated_at" timestamptz not null default now(), "deleted_at" timestamptz null, constraint "stripe_product_pkey" primary key ("id"));`);
    this.addSql(`CREATE INDEX IF NOT EXISTS "IDX_stripe_product_deleted_at" ON "stripe_product" (deleted_at) WHERE deleted_at IS NULL;`);
  }

  override async down(): Promise<void> {
    this.addSql(`drop table if exists "stripe_product" cascade;`);
  }

}
