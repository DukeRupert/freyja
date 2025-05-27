import { Migration } from '@mikro-orm/migrations';

export class Migration20250527025821 extends Migration {

  override async up(): Promise<void> {
    this.addSql(`create table if not exists "plan" ("id" text not null, "name" text not null, "active" boolean not null, "interval" text not null, "interval_count" integer not null, "created_at" timestamptz not null default now(), "updated_at" timestamptz not null default now(), "deleted_at" timestamptz null, constraint "plan_pkey" primary key ("id"));`);
    this.addSql(`CREATE INDEX IF NOT EXISTS "IDX_plan_deleted_at" ON "plan" (deleted_at) WHERE deleted_at IS NULL;`);
  }

  override async down(): Promise<void> {
    this.addSql(`drop table if exists "plan" cascade;`);
  }

}
