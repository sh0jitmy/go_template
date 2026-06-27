// Copyright 2026 [Copyright Holder]
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Author: [YOUR_NAME]

// Package database handles CGO-free sqlite initialization, database connection pools,
// schemas, automatic migrations, and data seeding using ent.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	entsql "entgo.io/ent/dialect/sql"
	sqlite "github.com/glebarez/go-sqlite"
	"github.com/shjtmy/go_sh0jitmy_template/ent"
	"github.com/shjtmy/go_sh0jitmy_template/ent/user"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	var found bool
	for _, d := range sql.Drivers() {
		if d == "sqlite3" {
			found = true
			break
		}
	}
	if !found {
		sql.Register("sqlite3", &sqlite.Driver{})
	}
}

// NewClient は SQLite データベースに接続し、接続プール設定が適用された ent.Client を返します。
func NewClient(ctx context.Context, dbPath string) (*ent.Client, error) {
	// DB接続の確立 (CGO-free "sqlite" ドライバーを使用)
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?cache=shared&_pragma=foreign_keys(1)", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// 接続プールの最適設定 (database-design:2 準拠)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(0) // インメモリDBやローカルの接続維持のため無制限、または適宜設定

	drv := entsql.OpenDB("sqlite3", db)
	client := ent.NewClient(ent.Driver(drv))

	// 自動マイグレーションの実行
	if err := client.Schema.Create(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to apply automatic schema migration: %w", err)
	}

	return client, nil
}

// SeedAdminUser はデータベースに初期管理者 (admin) ユーザーが存在しない場合、自動投入します。
func SeedAdminUser(ctx context.Context, client *ent.Client) error {
	exists, err := client.User.Query().
		Where(user.Username("admin")).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("failed to check admin user existence: %w", err)
	}

	if !exists {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin-pass"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash seed password: %w", err)
		}

		_, err = client.User.Create().
			SetUsername("admin").
			SetPasswordHash(string(hashedPassword)).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create seed admin user: %w", err)
		}
		slog.Info("Successfully seeded admin user in DB")
	}

	return nil
}
