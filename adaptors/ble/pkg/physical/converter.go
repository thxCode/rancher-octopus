package physical

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/rancher/octopus/adaptors/ble/api/v1alpha1"
)

// convertData helps to convert the data read from the device into meaningful data
func convertData(data []byte, dataConverter *v1alpha1.BluetoothDataConverter) (value, operatedValue string, err error) {
	var raw uint64
	raw, err = doBitsOperations(data, dataConverter)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to do bits operations")
	}
	value = strconv.FormatUint(raw, 10)

	operatedValue, err = doArithmeticOperations(float64(raw), dataConverter.OrderOfOperations)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to do arithmetic operations")
	}

	return value, operatedValue, nil
}

// doBitsOperations helps to calculate the data with bits converter.
func doBitsOperations(data []byte, bitsConverter *v1alpha1.BluetoothDataConverter) (uint64, error) {
	var (
		startIndex = bitsConverter.StartIndex
		endIndex   = bitsConverter.EndIndex
		shiftLeft  = bitsConverter.ShiftLeft
		shiftRight = bitsConverter.ShiftRight
	)

	var initialValue []byte
	if startIndex <= endIndex {
		initialValue = data[startIndex : endIndex+1]
	} else {
		for index := startIndex; index >= endIndex; index-- {
			initialValue = append(initialValue, data[index])
		}
	}

	var initialValueSb = &strings.Builder{}
	for _, value := range initialValue {
		initialValueSb.WriteString(strconv.Itoa(int(value)))
	}
	var initialByteValue, err = strconv.ParseUint(initialValueSb.String(), 16, 16)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse data to 16 bit size uint")
	}

	var result = initialByteValue
	if shiftLeft != 0 {
		result = result << shiftLeft
	} else if shiftRight != 0 {
		result = result >> shiftRight
	}
	return result, nil
}

// doArithmeticOperations helps to calculate the raw value with operations,
// and returns the calculated raw result in 6 digit precision.
func doArithmeticOperations(raw float64, operations []v1alpha1.BluetoothDeviceArithmeticOperation) (string, error) {
	if len(operations) == 0 {
		return "", nil
	}

	var result = raw
	for _, executeOperation := range operations {
		operationValue, err := strconv.ParseFloat(executeOperation.Value, 64)
		if err != nil {
			return "", errors.Wrapf(err, "failed to parse %s operation's value", executeOperation.Type)
		}
		switch executeOperation.Type {
		case v1alpha1.BluetoothDeviceArithmeticAdd:
			result = result + operationValue
		case v1alpha1.BluetoothDeviceArithmeticSubtract:
			result = result - operationValue
		case v1alpha1.BluetoothDeviceArithmeticMultiply:
			result = result * operationValue
		case v1alpha1.BluetoothDeviceArithmeticDivide:
			result = result / operationValue
		}
	}
	return strconv.FormatFloat(result, byte('f'), 6, 64), nil
}
