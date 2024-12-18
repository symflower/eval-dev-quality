package com.eval;

import org.junit.Test;
import static org.junit.Assert.assertEquals;

public class DecrementTest {
	@Test
	public void decrement() {
		int i = 1;
		int expected = 0;
		int actual = Decrement.decrement(i);

		assertEquals(expected, actual);
	}
}
