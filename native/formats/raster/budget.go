package raster

import "math"

// checkedAdd returns a+b or a RejectionDecodeBudget error on overflow.
func checkedAdd(a, b int64) (int64, error) {
	if b > 0 && a > math.MaxInt64-b {
		return 0, budgetOverflow()
	}
	if b < 0 && a < math.MinInt64-b {
		return 0, budgetOverflow()
	}
	return a + b, nil
}

// checkedMul returns a*b or a RejectionDecodeBudget error on overflow.
func checkedMul(a, b int64) (int64, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	product := a * b
	if product/b != a {
		return 0, budgetOverflow()
	}
	if a == -1 && b == math.MinInt64 {
		return 0, budgetOverflow()
	}
	if b == -1 && a == math.MinInt64 {
		return 0, budgetOverflow()
	}
	return product, nil
}

func budgetOverflow() error {
	return &RejectionError{RejectionDecodeBudget, "checked arithmetic overflow"}
}

// mipBudget computes base RGBA plus all ceil-halved mip levels.
func mipBudget(width, height int) (bytes int64, levels int, err error) {
	if width < 1 || height < 1 {
		return 0, 0, &RejectionError{
			RejectionDimensions,
			"mip dimensions must be positive",
		}
	}
	w, h := int64(width), int64(height)
	for {
		levelBytes, mulErr := checkedMul(w, h)
		if mulErr != nil {
			return 0, 0, mulErr
		}
		levelBytes, mulErr = checkedMul(levelBytes, 4)
		if mulErr != nil {
			return 0, 0, mulErr
		}
		bytes, err = checkedAdd(bytes, levelBytes)
		if err != nil {
			return 0, 0, err
		}
		levels++
		if w == 1 && h == 1 {
			break
		}
		w = (w + 1) / 2
		h = (h + 1) / 2
	}
	if bytes > MaxMipChainBytes {
		return 0, 0, &RejectionError{
			RejectionMipBudget,
			"RGBA mip chain exceeds 48MiB",
		}
	}
	return bytes, levels, nil
}

// decodeBudget applies the 128MiB modeled payload limit.
func decodeBudget(bytes int64) (int64, error) {
	if bytes > MaxDecodePayload {
		return 0, &RejectionError{
			RejectionDecodeBudget,
			"modeled decode payload exceeds 128MiB",
		}
	}
	return bytes, nil
}
