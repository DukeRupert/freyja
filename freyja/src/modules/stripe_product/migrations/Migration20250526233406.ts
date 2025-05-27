import { Migration } from '@mikro-orm/migrations';

export class Migration20250526233406 extends Migration {

  override async up(): Promise<void> {
    this.addSql(`alter table if exists "stripe_product" add column if not exists "stripe_id" text not null;`);
  }

  override async down(): Promise<void> {
    this.addSql(`alter table if exists "stripe_product" drop column if exists "stripe_id";`);
  }

}
