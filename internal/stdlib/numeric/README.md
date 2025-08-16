# numeric パッケージ概要

Orizon の数値標準ライブラリです。依存ライブラリなしで、ベクトル演算・行列演算・統計量・分解（LU/Cholesky）を提供します。

## 主な機能
- ベクトル演算: Dot/Axpy/Scale/Norm2
- 行列演算: MatVec/MatMul/Transpose、ユーティリティ（NewDense/FromRows/Eye/Zeros/Ones）
- 線形方程式: Solve（LU 分解ベース）、SolveSPD（Cholesky ベース）
- 行列分解: LUDecompose、Cholesky／Det／Inverse
- 統計: Mean/Variance/StdDev/Covariance/Correlation
- SIMD フレンドリーなコアカーネル（ループ展開）

## クイックスタート
```go
A := numeric.FromRows([][]float64{{4, 7}, {2, 6}})
// 行列式
_ = numeric.Det(A) // 10
// 連立一次方程式 Ax=b を解く
x := numeric.Solve(A, []float64{1, 0}) // [0.6, -0.2]

// SPD 行列の解法（Cholesky）
SPD := numeric.FromRows([][]float64{{4, 2}, {2, 3}})
x2 := numeric.SolveSPD(SPD, []float64{1, 1}) // [0.125, 0.25]

// 行列積
B := numeric.FromRows([][]float64{{1, 2}, {3, 4}})
C := numeric.MatMul(A, B)
_ = C.At(0, 0) // 4*1 + 7*3 = 25

// 統計
v := []float64{1, 2, 3, 4}
_ = numeric.Mean(v)      // 2.5
_ = numeric.Variance(v)  // 1.25（母分散でない定義の場合は実装に依存）
_ = numeric.StdDev(v)
```

## 注意事項
- 行列は行メジャーで内部表現されます。
- LU は部分ピボット選択付き。特異行列では `nil` を返します。
- Cholesky は SPD 判定をし、非 SPD の場合は `nil` を返します。
- パフォーマンスを重視しつつ、見通しのよい実装を優先しています（今後、ブロッキングや並列化の拡張余地あり）。

## テスト
`go test ./internal/stdlib/numeric -v` でユニットテストを実行できます。
