package com.eval;

import org.junit.Test;
import static org.junit.Assert.assertEquals;

public class IncrementTest {
	@Test
	public void increment() {
		int i = 1;
		int expected = 2;
		int actual = Increment.increment(i);

		assertEquals(expected, actual);
	}
}
