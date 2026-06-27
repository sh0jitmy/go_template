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

package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shjtmy/go_sh0jitmy_template/ent"
)

// SetupEngine は Gin エンジンを構成し、ミドルウェアおよびハンドラーを登録します。
func SetupEngine(db *ent.Client) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// HSTSミドルウェアを全体に適用
	r.Use(HSTSSetMiddleware())

	// OTelメトリクス収集ミドルウェアを全体に適用
	r.Use(OTelMetricsMiddleware())

	s := NewServer(db)

	// メトリクススクレイプ用エンドポイント（Prometheus形式でエクスポート）
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 手動マッピング（OpenAPI自動生成された ServerInterface 実装のバインド）
	r.POST("/login", func(c *gin.Context) {
		s.Login(c)
	})

	authorized := r.Group("/")
	authorized.Use(BearerAuthMiddleware("secret-bearer-token"))
	authorized.GET("/users/me", func(c *gin.Context) {
		s.GetMe(c)
	})

	return r
}
