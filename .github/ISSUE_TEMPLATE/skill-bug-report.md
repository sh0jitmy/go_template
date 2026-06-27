name: 💡 スキル不適合・バグ報告 (Skill Bug Report)
description: スキルの誤動作（反応しない、または誤反応）、誤ったコード提案などの改善提案を行います。
title: "[BUG/RF]: スキル名 - 短い要約"
labels: ["bug", "skill-improvement"]
body:
  - type: markdown
    attributes:
      value: |
        スキルファイルの改善やバグ修正について教えてください。
  - type: dropdown
    id: skill-name
    attributes:
      label: 🎯 対象のスキル名
      description: 修正が必要なカスタム指示（Skill）を選択してください。
      options:
        - database-design
        - sre-deployment
        - agent-skill-evaluator
        - golang-design
        - golang-implementation
        - golang-observability
        - golang-e2e-testing
    validations:
      required: true
  - type: dropdown
    id: bug-type
    attributes:
      label: 🚨 問題の種類
      description: どのような不適合が発生しましたか？
      options:
        - "Under-triggering (必要な会話でスキルがロードされなかった)"
        - "Over-triggering (無関係な会話でスキルが誤ってロードされた)"
        - "Incorrect/Bad Advice (スキルが誤った設計・実装を提案した)"
        - "Missing Rule (不足している重要なプラクティスがある)"
    validations:
      required: true
  - type: textarea
    id: description
    attributes:
      label: 📝 問題の具体的な内容
      description: どのような指示（プロンプト）の際に、どのような問題が発生したかを記述してください。
      placeholder: |
        指示プロンプト: 「〜の並行処理を書いて」
        発生した挙動: エージェントがクローズ漏れのあるチャネルコードを提案した。
    validations:
      required: true
  - type: textarea
    id: expected-behavior
    attributes:
      label: 💡 期待される挙動 / あるべき指針
      description: 本来、スキルがどのようにエージェントを誘導すべきだったかを記述してください。
      placeholder: |
        期待される挙動: チャネルの所有者がクローズすることを明記し、goleak等を用いたテスト作成を促すべき。
    validations:
      required: true
  - type: textarea
    id: proposed-eval-test
    attributes:
      label: 🧪 評価用アサーション (Evals定義案)
      description: この問題を検知するための評価（Evals）のインプットプロンプトとアサーション（判定KW）の案があれば教えてください。
      placeholder: |
        Prompt: 「〜の処理を並行化して」
        Assertions: "channel close", "goleak", "close"
